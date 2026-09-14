// Package webapp — статика Telegram Mini App, встроенная в бинарник.
// Файлы лежат в internal/webapp/static и компилируются в proakt:
// один бинарник = бот + API + веб-приложение (как и весь проект, без Node).
package webapp

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:static
var files embed.FS

// Handler — HTTP-обработчик статики.
// index.html — всегда no-cache (обновляется с релизом), ассеты — с коротким
// кэшем: файлы маленькие, а единственный пользователь — мастер с телефоном.
func Handler() http.Handler {
	sub, err := fs.Sub(files, "static")
	if err != nil {
		panic("webapp: встроенная статика недоступна: " + err.Error())
	}
	fileServer := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" || !fsValid(sub, p) {
			r.URL.Path = "/" // SPA-фолбэк: неизвестные пути → index.html
			p = "index.html"
		}
		if p == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		fileServer.ServeHTTP(w, r)
	})
}

func fsValid(fsys fs.FS, name string) bool {
	st, err := fs.Stat(fsys, name)
	return err == nil && !st.IsDir()
}
