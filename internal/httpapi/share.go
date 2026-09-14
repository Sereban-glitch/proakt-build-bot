// share.go — публичный просмотр сметы заказчиком (v0.5).
//
// Заказчик не в боте и не должен ничего устанавливать: мастер жмёт «Поделиться»,
// присылает ссылку /s/<токен> — и в любом браузере открывается аккуратная
// смета: чистовые работы, скрытая подготовка (с объяснением, зачем), фото
// скрытых работ и итоги. Доступ по криптостойкому токену, без авторизации.
package httpapi

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proakt/internal/domain"
)

const shareTmplSrc = `<!DOCTYPE html>
<html lang="ru"><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="robots" content="noindex,nofollow">
<title>{{.Title}} — смета</title>
<style>
:root{
  --bg:#0f1620;--card:rgba(255,255,255,.055);--border:rgba(255,255,255,.1);
  --text:#eef2f6;--sub:#a7b4c2;--hint:#7c8894;
  --ok:#2ecc71;--warn:#f1c40f;--accent:#38bdf8;--hidden-bg:rgba(241,196,15,.08);--hidden-bd:rgba(241,196,15,.22);
}
@media (prefers-color-scheme: light){
  :root{--bg:#eef2f7;--card:#ffffff;--border:rgba(15,40,70,.1);--text:#17212b;--sub:#5a6a7a;--hint:#8a97a5;
        --hidden-bg:rgba(241,196,15,.14);--hidden-bd:rgba(200,160,10,.25);}
}
*{box-sizing:border-box;margin:0;padding:0}
body{font:15px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Arial,sans-serif;
     background:var(--bg);color:var(--text);padding:20px 14px 60px;
     background-image:radial-gradient(1200px 500px at 50% -200px, rgba(56,189,248,.08), transparent);}
.wrap{max-width:640px;margin:0 auto}
h1{font-size:20px;line-height:1.3;margin-bottom:4px}
.obj{color:var(--sub);font-size:13.5px;margin-bottom:14px}
.badge{display:inline-block;font-size:11.5px;font-weight:600;padding:3px 10px;border-radius:999px;
       border:1px solid var(--border);color:var(--sub);background:var(--card);vertical-align:middle;margin-left:6px}
.badge.s-sent{color:#f1c40f}.badge.s-approved{color:#2ecc71}.badge.s-done{color:#38bdf8}
.totals{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin:16px 0}
.t{background:var(--card);border:1px solid var(--border);border-radius:14px;padding:10px 12px;backdrop-filter:blur(14px)}
.t .l{font-size:11px;color:var(--hint);margin-bottom:2px}
.t .v{font-size:15.5px;font-weight:700;font-variant-numeric:tabular-nums}
.t.hi .v{color:var(--warn)}
section{margin-top:18px}
h2{font-size:13px;text-transform:uppercase;letter-spacing:.06em;color:var(--hint);margin-bottom:8px}
.card{background:var(--card);border:1px solid var(--border);border-radius:16px;overflow:hidden;backdrop-filter:blur(14px)}
.row{display:grid;grid-template-columns:1fr auto;gap:2px 12px;padding:11px 14px;border-bottom:1px solid var(--border)}
.row:last-child{border-bottom:0}
.row .n{font-size:14px}
.row .m{font-size:12px;color:var(--hint)}
.row .s{font-size:14px;font-weight:600;text-align:right;font-variant-numeric:tabular-nums;white-space:nowrap}
.row.hiddenw{background:var(--hidden-bg)}
.row.hiddenw .n::before{content:"▪ ";color:var(--warn)}
.note{font-size:12px;color:var(--sub);grid-column:1/-1}
.gtotal{display:flex;justify-content:space-between;align-items:center;padding:14px;
        background:linear-gradient(135deg, rgba(46,204,113,.12), rgba(56,189,248,.10));
        border:1px solid rgba(46,204,113,.3);border-radius:16px;margin-top:14px}
.gtotal b{font-size:18px;font-variant-numeric:tabular-nums}
.photos{display:grid;grid-template-columns:repeat(3,1fr);gap:8px}
.photos img{width:100%;aspect-ratio:1;object-fit:cover;border-radius:12px;border:1px solid var(--border);cursor:zoom-in}
.foot{margin-top:22px;color:var(--hint);font-size:11.5px;text-align:center;line-height:1.6}
.tip{margin-top:8px;padding:10px 12px;border:1px dashed var(--hidden-bd);border-radius:12px;background:var(--hidden-bg);
     color:var(--sub);font-size:12.5px;line-height:1.5}
#lb{position:fixed;inset:0;background:rgba(5,10,16,.88);display:none;align-items:center;justify-content:center;z-index:9;cursor:zoom-out}
#lb img{max-width:96vw;max-height:92vh;border-radius:10px}
.approve{background:linear-gradient(135deg, rgba(46,204,113,.14), rgba(56,189,248,.10));border:1px solid rgba(46,204,113,.35);
         border-radius:16px;padding:16px;margin-top:4px}
.approve-t{font-weight:700;font-size:15px;margin-bottom:4px}
.approve-s{font-size:12.5px;color:var(--sub);margin-bottom:12px}
.appr-btn{width:100%;border:0;border-radius:12px;padding:13px;font:inherit;font-weight:700;font-size:15px;cursor:pointer;
          background:#2ecc71;color:#06281a}
.appr-btn:active{transform:scale(.98)}
.cmt .n{color:var(--accent)}
.cmt.mine .n{color:var(--ok)}
.cmt-form{display:flex;gap:8px;margin-top:10px}
.cmt-form textarea{flex:1;background:var(--card);border:1px solid var(--border);border-radius:12px;color:var(--text);
                   font:inherit;font-size:14px;padding:10px 12px;resize:none;min-height:44px}
.cmt-send{border:0;border-radius:12px;padding:0 16px;font:inherit;font-weight:600;cursor:pointer;
          background:var(--accent);color:#06243a}
.print-link{margin-top:14px;text-align:center;font-size:13px}
.print-link a{color:var(--accent);text-decoration:none}
</style></head><body>
<div class="wrap">
  <h1>{{.Title}}<span class="badge {{StatusClass .Status}}">{{StatusText .Status}}</span></h1>
  <div class="obj">Объект: <b>{{.ObjectName}}</b></div>

  {{if .Note}}<div class="tip">{{.Note}}</div>{{end}}

  <div class="totals">
    <div class="t"><div class="l">Чистовые работы</div><div class="v">{{Money .Visible}}</div></div>
    <div class="t hi"><div class="l">Скрытая подготовка</div><div class="v">{{Money .Hidden}}</div></div>
    <div class="t"><div class="l">Всего по смете</div><div class="v">{{Money .Total}}</div></div>
  </div>

  <section>
    <h2>Позиции сметы</h2>
    <div class="card">
      {{range .Lines}}
      <div class="row{{if .Hidden}} hiddenw{{end}}">
        <div class="n">{{.Name}}</div>
        <div class="s">{{Money .Sum}}</div>
        <div class="m">{{QtyText .}}{{if .Done}} · <span style="color:var(--ok)">✓ закрыто актом</span>{{end}}</div>
        {{if .Note}}<div class="note">{{.Note}}</div>{{end}}
      </div>
      {{end}}
    </div>
    {{if gt .HiddenShare 0}}
    <div class="tip">▪ <b>Скрытые работы</b> — подготовительные этапы перед чистовой отделкой
    (грунтовка, армирование, шпаклёвка, гидроизоляция). Они не видны в готовом ремонте,
    но именно они дают прочность и ровность — и занимают <b>{{.HiddenShare}}%</b> сметы.
    Каждое фото ниже — подтверждение выполненных этапов.</div>
    {{end}}
    <div class="gtotal"><span>Итого по смете</span><b>{{Money .Total}}</b></div>
  </section>

  {{if .Photos}}
  <section>
    <h2>Фото скрытых работ</h2>
    <div class="photos">
      {{range .Photos}}<img loading="lazy" src="photo/{{.ID}}" alt="{{.Caption}}" onclick="lb(this)">{{end}}
    </div>
  </section>
  {{end}}

  {{/* --- v0.6: согласование и диалог --- */}}
  {{if .CanApprove}}
  <section>
    <div class="approve">
      <div class="approve-t">Смета вас устраивает?</div>
      <div class="approve-s">Нажмите — мастер сразу получит уведомление и приступит к закупкам/работам.</div>
      <button id="appr-btn" class="appr-btn" onclick="clientAction('approve', {}, this)">✅ Принять смету</button>
    </div>
  </section>
  {{end}}

  <section>
    <h2>Вопросы и комментарии</h2>
    {{if .Comments}}
    <div class="card">
      {{range .Comments}}
      <div class="row cmt{{if eq .Author "client"}} mine{{end}}">
        <div class="n">{{if eq .Author "client"}}Вы{{else}}Мастер{{end}}</div>
        <div class="s">{{TimeShort .CreatedAt}}</div>
        <div class="note">{{.Text}}</div>
      </div>
      {{end}}
    </div>
    {{else}}
    <div class="tip">Пока без комментариев — напишите вопрос мастеру ниже (например: «можно ли поменять плитку на другую?»).</div>
    {{end}}
    <div class="cmt-form">
      <textarea id="cmt-text" maxlength="500" rows="2" placeholder="Ваш вопрос или комментарий…"></textarea>
      <button id="cmt-send" class="cmt-send" onclick="sendComment()">Отправить</button>
    </div>
  </section>

  <div class="print-link"><a href="print">🖨 Печать / сохранить в PDF</a></div>

  <div class="foot">Смета сформирована приложением «ПрорАКТ» · мастер ведёт учёт работ и денег<br>
  Суммы указаны с учётом сложности работ. Вопросы — пишите мастеру в Telegram.</div>
</div>
<div id="lb" onclick="this.style.display='none'"><img id="lbimg" alt=""></div>
<script>
function lb(im){var l=document.getElementById('lb');document.getElementById('lbimg').src=im.src;l.style.display='flex';}

// --- v0.6: согласование и диалог (токен — из адреса страницы) ---
var TOKEN = location.pathname.split('/')[2];

function clientAction(action, body, btn){
  btn.disabled = true;
  fetch('/s/' + TOKEN + '/' + action, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(body || {})
  }).then(function(r){
    return r.json().then(function(j){ return {ok: r.ok, j: j}; });
  }).then(function(res){
    if (res.ok) {
      if (action === 'approve') {
        alert('Спасибо! Смета отмечена как согласованная — мастер получил уведомление.');
        location.reload();
      } else {
        location.reload();
      }
    } else {
      alert((res.j && res.j.error) || 'Не получилось');
      btn.disabled = false;
    }
  }).catch(function(){ alert('Нет связи — попробуйте ещё раз'); btn.disabled = false; });
}

function sendComment(){
  var t = document.getElementById('cmt-text');
  if (!t.value.trim()) { t.focus(); return; }
  clientAction('comment', {text: t.value.trim()}, document.getElementById('cmt-send'));
}
</script>
</body></html>`

var shareTmpl = template.Must(template.New("share").Funcs(template.FuncMap{
	// Money — 12 345,50 грн (пробел-разделитель, запятая — как в таблицах)
	"Money": func(v float64) string {
		neg := v < 0
		if neg {
			v = -v
		}
		k := int64(v*100+0.5) / 100
		kop := int64(v*100+0.5) % 100
		s := fmtInt(k)
		var parts []string
		for len(s) > 3 {
			parts = append([]string{s[len(s)-3:]}, parts...)
			s = s[:len(s)-3]
		}
		parts = append([]string{s}, parts...)
		out := strings.Join(parts, "\u2009")
		if kop > 0 {
			out += "," + pad2(kop)
		}
		if neg {
			out = "-" + out
		}
		return out + " грн"
	},
	"QtyText": func(l domain.EstimateLine) string {
		q := trimZero(l.Qty)
		if l.Unit == "" {
			if l.Qty > 0 {
				return q + " × " + trimZero(l.Price) + " грн"
			}
			return trimZero(l.Price) + " грн (за всё)"
		}
		if l.Qty > 0 {
			return q + " " + l.Unit + " × " + trimZero(l.Price) + " грн"
		}
		return trimZero(l.Price) + " грн/" + l.Unit
	},
	"StatusText": func(s string) string {
		switch s {
		case "sent":
			return "отправлена"
		case "approved":
			return "согласована"
		case "done":
			return "закрыта"
		default:
			return "черновик"
		}
	},
	"StatusClass": func(s string) string {
		switch s {
		case "sent", "approved", "done":
			return "s-" + s
		}
		return ""
	},
	// v0.6: «31 авг 15:04» для комментариев
	"TimeShort": func(t time.Time) string {
		return t.Format("02.01 15:04")
	},
	"EqAuthor": func(a, b string) bool { return a == b },
}).Parse(shareTmplSrc))

func fmtInt(v int64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

func pad2(v int64) string {
	if v < 10 {
		return "0" + fmtInt(v)
	}
	return fmtInt(v)
}

func trimZero(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// handleSharePage — GET /s/{token}: страница сметы для заказчика.
func (s *Server) handleSharePage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if len(token) < 8 || len(token) > 64 || !isTokenSafe(token) {
		http.NotFound(w, r)
		return
	}
	v, err := s.svc.GetShareView(r.Context(), token)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="ru"><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>body{font:15px/1.6 -apple-system,sans-serif;background:#0f1620;color:#eef2f6;display:flex;min-height:90vh;align-items:center;justify-content:center;text-align:center;padding:20px}</style>
<title>Смета не найдена</title><div><div style="font-size:44px">🧾</div><h1 style="font-size:18px">Смета не найдена</h1>
<p style="color:#7c8894">Ссылка устарела или была перевыпущена мастером.<br>Попросите свежую ссылку в чате.</p></div>`))
		return
	}
	// деньги: считаем из строк с коэффициентом из первой строки? — хранится в смете,
	// ShareView уже содержит линии; коэффициент нужен из estimates — дозаполняем totals:
	var total, visible, hidden float64
	for _, l := range v.Lines {
		if l.Hidden {
			hidden += l.Sum
		} else {
			visible += l.Sum
		}
		total += l.Sum
	}
	v.Total, v.Visible, v.Hidden = total, visible, hidden
	done := 0
	for _, l := range v.Lines {
		if l.Done {
			done++
		}
	}
	if total > 0.01 {
		v.HiddenShare = int(hidden/total*100 + 0.5)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	_ = shareTmpl.Execute(w, &v)
}

// handleSharePhoto — GET /s/{token}/photo/{id}: снимок объекта этой сметы.
// path traversal исключён той же проверкой, что и в handlePhotoFile.
func (s *Server) handleSharePhoto(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	id, ok := pathID(r, "id")
	if !ok || len(token) < 8 || !isTokenSafe(token) {
		http.NotFound(w, r)
		return
	}
	path, err := s.svc.SharePhotoFile(r.Context(), token, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	root, _ := filepath.Abs(s.cfg.FilesDir)
	full, err := filepath.Abs(path)
	if err != nil || !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, full)
}

func isTokenSafe(s string) bool {
	for _, c := range s {
		ok := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
		if !ok {
			return false
		}
	}
	return true
}
