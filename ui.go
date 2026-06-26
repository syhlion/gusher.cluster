package main

import "net/http"

// uiHTML is a single-page engineering console served by the master. It polls the
// read-only observability endpoints — global stats, per-app breakdown, and (for
// a given app_key) channels with their online counts. Not an end-user product —
// a probe + API-validation surface for RD/ops, matching gua's console.
const uiHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>gusher console</title>
<style>
 body{font:13px/1.4 monospace;margin:16px;background:#0b0e14;color:#c9d1d9}
 h1{font-size:16px} h2{font-size:13px;color:#7ee787;margin:16px 0 6px}
 table{border-collapse:collapse;width:100%;margin-bottom:8px}
 td,th{border:1px solid #2d333b;padding:3px 6px;text-align:left;font-size:12px}
 th{background:#161b22;color:#8b949e}
 .ok{color:#3fb950} .bad{color:#f85149} .muted{color:#8b949e}
 input,button{font:12px monospace;background:#161b22;color:#c9d1d9;border:1px solid #2d333b;padding:3px 6px}
 #bar{margin-bottom:10px} b{color:#e6edf3}
</style></head><body>
<h1>gusher console <span class="muted" id="ver"></span> <span id="health"></span></h1>
<div id="bar">app_key: <input id="app" placeholder="app key" size="16">
 <button onclick="load()">load</button>
 <button onclick="toggle()">auto-refresh: <span id="ar">off</span></button></div>
<h2>global</h2><div id="global"></div>
<h2>apps</h2><div id="apps"></div>
<h2>channels <span class="muted" id="appname"></span></h2><div id="channels"></div>
<script>
let timer=null;
const esc=s=>String(s==null?'':s).replace(/[&<>]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;'}[c]));
async function j(u){const r=await fetch(u);if(!r.ok)throw new Error(u+' '+r.status);return r.json();}
async function ok(u){try{const r=await fetch(u);return r.ok;}catch(e){return false;}}
function table(rows,cols){if(!rows||!rows.length)return '<div class="muted">(none)</div>';
 let h='<table><tr>'+cols.map(c=>'<th>'+c[0]+'</th>').join('')+'</tr>';
 for(const x of rows)h+='<tr>'+cols.map(c=>'<td>'+c[1](x)+'</td>').join('')+'</tr>';return h+'</table>';}
async function load(){
 try{document.getElementById('ver').textContent=await j('/version');}catch(e){}
 // health row
 const live=await ok('/healthz'), ready=await ok('/readyz');
 document.getElementById('health').innerHTML=
  '· master: '+(live?'<span class=ok>● healthy</span>':'<span class=bad>● down</span>')+
  ' &nbsp; ready: '+(ready?'<span class=ok>● ok</span>':'<span class=bad>● not ready</span>');
 // global totals
 try{const s=await j('/v1/stats');
  document.getElementById('global').innerHTML=
   '<div>apps: <b>'+s.apps+'</b> &nbsp; connections: <b>'+s.connections+'</b>'+
   ' &nbsp; online users: <b>'+s.users+'</b> <span class=muted>(users approx)</span></div>';
 }catch(e){document.getElementById('global').innerHTML='<span class=bad>'+esc(e.message)+'</span>';}
 // per-app breakdown
 try{const apps=await j('/v1/apps');
  document.getElementById('apps').innerHTML=table(apps,[['app',x=>esc(x.app)],
   ['connections',x=>x.connections],['online users',x=>x.users]]);
 }catch(e){document.getElementById('apps').innerHTML='<span class=bad>'+esc(e.message)+'</span>';}
 // channels for the entered app_key
 const g=document.getElementById('app').value.trim();
 document.getElementById('appname').textContent=g?('('+g+')'):'';
 if(!g){document.getElementById('channels').innerHTML='<div class=muted>enter an app_key</div>';return;}
 try{const chs=await j('/v1/apps/'+encodeURIComponent(g)+'/channels');
  const rows=await Promise.all((chs||[]).map(async ch=>{
   let n='?';try{n=(await j('/v1/apps/'+encodeURIComponent(g)+'/channels/'+encodeURIComponent(ch)+'/users/count')).count;}catch(e){}
   return {channel:ch,online:n};}));
  document.getElementById('channels').innerHTML=table(rows,[['channel',x=>esc(x.channel)],['online',x=>x.online]]);
 }catch(e){document.getElementById('channels').innerHTML='<span class=bad>'+esc(e.message)+'</span>';}
}
function toggle(){if(timer){clearInterval(timer);timer=null;document.getElementById('ar').textContent='off';}
 else{timer=setInterval(load,3000);document.getElementById('ar').textContent='on';load();}}
load();
</script></body></html>`

func UI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(uiHTML))
	}
}
