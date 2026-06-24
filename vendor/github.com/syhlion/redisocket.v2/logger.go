package redisocket

import (
	"errors"
	"io"
	"log/slog"
	"os"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// LogOutput 決定 log 寫到哪。
type LogOutput int

const (
	LogStdout LogOutput = iota // 只寫 stdout
	LogFile                    // 只寫檔案（含輪替）
	LogBoth                    // 同時寫 stdout 與檔案（io.MultiWriter）
)

// LogConfig 設定要建立的 logger。
// 引擎核心（NewHub…）只吃 *slog.Logger、不管輸出;NewLogger 是 opt-in 的便利
// 建構器,把「輸出目的地 + 檔案輪替」這件事(app 的職責)集中成可設定選項。
type LogConfig struct {
	Output LogOutput  // stdout / file / both
	File   string     // LogFile / LogBoth 時的檔案路徑
	Format string     // "json"(預設) 或 "text"
	Level  slog.Level // slog.LevelInfo / LevelDebug …

	// 檔案輪替（logrotate;僅 LogFile / LogBoth 生效,零值用合理預設）。
	// 由 lumberjack 處理:依大小輪替、保留份數/天數、可壓縮。
	MaxSizeMB  int  // 單檔上限 MB(達到即輪替),預設 100
	MaxBackups int  // 最多保留幾個舊檔,預設 7
	MaxAgeDays int  // 舊檔最多保留幾天,預設 30
	Compress   bool // 舊檔是否 gzip 壓縮
}

// NewLogger 依設定建立 *slog.Logger,回傳 logger 與一個 close 函式
// (關閉檔案/輪替器;stdout-only 時為 no-op,呼叫端在 shutdown 時呼叫)。
//
// 範例(寫 stdout＋檔案、檔案達 100MB 輪替、留 7 份、壓縮):
//
//	lg, closeLog, err := redisocket.NewLogger(redisocket.LogConfig{
//	    Output: redisocket.LogBoth, File: "/var/log/gusher.log",
//	    Format: "json", Level: slog.LevelInfo,
//	    MaxSizeMB: 100, MaxBackups: 7, MaxAgeDays: 30, Compress: true,
//	})
//	defer closeLog()
//	hub := redisocket.NewHubWithBrokerAndPresence(broker, presence, lg, false)
func NewLogger(cfg LogConfig) (*slog.Logger, func() error, error) {
	noop := func() error { return nil }

	var (
		w      io.Writer
		closer = noop
	)
	switch cfg.Output {
	case LogStdout:
		w = os.Stdout
	case LogFile:
		lj, err := newRotatingWriter(cfg)
		if err != nil {
			return nil, nil, err
		}
		w = lj
		closer = lj.Close
	case LogBoth:
		lj, err := newRotatingWriter(cfg)
		if err != nil {
			return nil, nil, err
		}
		w = io.MultiWriter(os.Stdout, lj) // 兩者都要
		closer = lj.Close
	default:
		return nil, nil, errors.New("redisocket: unknown LogOutput")
	}

	opts := &slog.HandlerOptions{Level: cfg.Level}
	var h slog.Handler
	if cfg.Format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(h), closer, nil
}

// newRotatingWriter 回傳一個會自動輪替的檔案 writer(lumberjack）。
func newRotatingWriter(cfg LogConfig) (*lumberjack.Logger, error) {
	if cfg.File == "" {
		return nil, errors.New("redisocket: LogConfig.File required for file output")
	}
	return &lumberjack.Logger{
		Filename:   cfg.File,
		MaxSize:    orDefault(cfg.MaxSizeMB, 100),
		MaxBackups: orDefault(cfg.MaxBackups, 7),
		MaxAge:     orDefault(cfg.MaxAgeDays, 30),
		Compress:   cfg.Compress,
	}, nil
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
