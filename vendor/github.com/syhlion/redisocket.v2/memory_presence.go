package redisocket

import (
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// memoryPresence 是 NATS-native 的 Presence 實作:每個節點只在「記憶體」保存
// 自己持有的連線;跨節點查詢透過 NATS request/reply 即時 scatter-gather 聚合,
// 不需要任何共享 store、也沒有 KEYS。
//
// 語意與 redisPresence 不同:presence 從「全域共享 store(強一致)」轉為
// 「查詢時即時詢問所有節點(最終一致)」。Touch/Sync 只更新本節點狀態。
type memoryPresence struct {
	nc           *nats.Conn
	prefix       string
	queryTimeout time.Duration
	sub          *nats.Subscription

	mu sync.RWMutex
	// appKey -> set(uid)
	online map[string]map[string]struct{}
	// appKey -> channel -> set(uid)
	channels map[string]map[string]map[string]struct{}
}

// presenceQuery / presenceReply 是跨節點查詢的 envelope(走 NATS request/reply)。
type presenceQuery struct {
	Type    string `json:"t"` // online | channel | channels
	AppKey  string `json:"a"`
	Channel string `json:"c"`
	Pattern string `json:"p"`
}
type presenceReply struct {
	Uids     []string `json:"u,omitempty"`
	Channels []string `json:"c,omitempty"`
}

const defaultPresenceQueryTimeout = 100 * time.Millisecond

// newMemoryPresence 建立 memoryPresence 並啟動「回應其他節點查詢」的訂閱。
func NewMemoryPresence(nc *nats.Conn, prefix string) (Presence, error) {
	p := &memoryPresence{
		nc:           nc,
		prefix:       prefix,
		queryTimeout: defaultPresenceQueryTimeout,
		online:       make(map[string]map[string]struct{}),
		channels:     make(map[string]map[string]map[string]struct{}),
	}
	// 每個節點都訂閱查詢主題,回覆自己本機的成員。用 queue group 之外的純 fan-out:
	// 不設 queue,故所有節點都會收到並回覆(scatter-gather)。
	sub, err := nc.Subscribe(p.querySubject(), p.handleQuery)
	if err != nil {
		return nil, err
	}
	p.sub = sub
	return p, nil
}

func (p *memoryPresence) querySubject() string {
	return p.prefix + "presence.query"
}

// ---- 本機狀態更新(Presence 介面) ----

func (p *memoryPresence) Touch(prefix, appKey, uid string, chs []string) error {
	if uid == "" && len(chs) == 0 {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.addLocked(appKey, uid, chs)
	return nil
}

// Sync 是週期性的完整快照:以 members 重建本節點狀態(自然處理已離線成員的移除)。
func (p *memoryPresence) Sync(prefix string, members []PresenceMember) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.online = make(map[string]map[string]struct{})
	p.channels = make(map[string]map[string]map[string]struct{})
	for _, m := range members {
		p.addLocked(m.AppKey, m.Uid, m.Channels)
	}
	return nil
}

func (p *memoryPresence) addLocked(appKey, uid string, chs []string) {
	if uid != "" {
		if p.online[appKey] == nil {
			p.online[appKey] = make(map[string]struct{})
		}
		p.online[appKey][uid] = struct{}{}
	}
	for _, ch := range chs {
		if p.channels[appKey] == nil {
			p.channels[appKey] = make(map[string]map[string]struct{})
		}
		if p.channels[appKey][ch] == nil {
			p.channels[appKey][ch] = make(map[string]struct{})
		}
		if uid != "" {
			p.channels[appKey][ch][uid] = struct{}{}
		}
	}
}

// ---- 查詢(本機回覆 + 跨節點聚合) ----

func (p *memoryPresence) Online(prefix, appKey string) ([]string, error) {
	return p.scatterUids(presenceQuery{Type: "online", AppKey: appKey})
}

func (p *memoryPresence) OnlineByChannel(prefix, appKey, channel string) ([]string, error) {
	return p.scatterUids(presenceQuery{Type: "channel", AppKey: appKey, Channel: channel})
}

func (p *memoryPresence) Channels(prefix, appKey, pattern string) ([]string, error) {
	q := presenceQuery{Type: "channels", AppKey: appKey, Pattern: pattern}
	replies, err := p.scatter(q)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	for _, r := range replies {
		for _, c := range r.Channels {
			set[c] = struct{}{}
		}
	}
	return setToSlice(set), nil
}

func (p *memoryPresence) scatterUids(q presenceQuery) ([]string, error) {
	replies, err := p.scatter(q)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	for _, r := range replies {
		for _, u := range r.Uids {
			set[u] = struct{}{}
		}
	}
	return setToSlice(set), nil
}

// scatter 發出查詢、在 queryTimeout 內收集所有節點(含自己)的回覆。
func (p *memoryPresence) scatter(q presenceQuery) ([]presenceReply, error) {
	data, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}
	inbox := p.nc.NewRespInbox()
	sub, err := p.nc.SubscribeSync(inbox)
	if err != nil {
		return nil, err
	}
	defer sub.Unsubscribe()
	if err := p.nc.PublishRequest(p.querySubject(), inbox, data); err != nil {
		return nil, err
	}
	var replies []presenceReply
	deadline := time.Now().Add(p.queryTimeout)
	for {
		d := time.Until(deadline)
		if d <= 0 {
			break
		}
		msg, err := sub.NextMsg(d)
		if err != nil {
			break // 逾時:收集結束
		}
		var r presenceReply
		if json.Unmarshal(msg.Data, &r) == nil {
			replies = append(replies, r)
		}
	}
	return replies, nil
}

// handleQuery 回覆本節點記憶體中符合的成員。
func (p *memoryPresence) handleQuery(m *nats.Msg) {
	var q presenceQuery
	if err := json.Unmarshal(m.Data, &q); err != nil {
		return
	}
	var reply presenceReply
	p.mu.RLock()
	switch q.Type {
	case "online":
		for uid := range p.online[q.AppKey] {
			reply.Uids = append(reply.Uids, uid)
		}
	case "channel":
		for uid := range p.channels[q.AppKey][q.Channel] {
			reply.Uids = append(reply.Uids, uid)
		}
	case "channels":
		for ch := range p.channels[q.AppKey] {
			if globMatch(q.Pattern, ch) {
				reply.Channels = append(reply.Channels, ch)
			}
		}
	}
	p.mu.RUnlock()
	b, err := json.Marshal(reply)
	if err != nil {
		return
	}
	m.Respond(b)
}

// Close 停止回應訂閱。
func (p *memoryPresence) Close() error {
	if p.sub != nil {
		return p.sub.Unsubscribe()
	}
	return nil
}

func setToSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// globMatch 支援單一 "*" 萬用(對齊 redisPresence 的 KEYS pattern 常見用法:
// "*" = 全部、"abc*" = 前綴、"*abc" = 後綴、其餘為精確比對)。
func globMatch(pattern, s string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	star := strings.IndexByte(pattern, '*')
	if star < 0 {
		return pattern == s
	}
	pre := pattern[:star]
	suf := pattern[star+1:]
	return len(s) >= len(pre)+len(suf) && strings.HasPrefix(s, pre) && strings.HasSuffix(s, suf)
}
