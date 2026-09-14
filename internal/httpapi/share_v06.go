// share_v06.go — действия заказчика на публичной странице (v0.6):
//
//	POST /s/{token}/approve — «✅ Принять смету» → статус approved + уведомление мастеру в чат
//	POST /s/{token}/comment — вопрос/комментарий заказчика → в диалог сметы + уведомление
//	GET  /s/{token}/print   — печатная версия (браузер → «Сохранить как PDF», без зависимостей)
//
// Уведомления — через Notifier (интерфейс): main.go подключает бота,
// тесты — заглушку. Никакого цикла httpapi → app.
package httpapi

import (
	"log"
	"net/http"
	"strings"

	"proakt/internal/store"
)

// Notifier — уведомление мастеру о действиях заказчика (v0.6).
type Notifier interface {
	NotifyEstimateApproved(chatID int64, title string)
	NotifyClientComment(chatID int64, title, text string)
}

// noopNotifier — когда бот не подключён (тесты/отдельный веб-сервер).
type noopNotifier struct{}

func (noopNotifier) NotifyEstimateApproved(int64, string)      {}
func (noopNotifier) NotifyClientComment(int64, string, string) {}

// SetNotifier — подключение бота как источника уведомлений (из main.go).
func (s *Server) SetNotifier(n Notifier) {
	if n != nil {
		s.notifier = n
	}
}

// notify — уведомление не должно ломать запрос заказчика при падении бота.
func (s *Server) notify(f func(Notifier)) {
	defer func() { _ = recover() }()
	f(s.notifier)
}

func tokenOK(w http.ResponseWriter, r *http.Request) (string, bool) {
	token := r.PathValue("token")
	if len(token) < 8 || len(token) > 64 || !isTokenSafe(token) {
		http.NotFound(w, r)
		return "", false
	}
	return token, true
}

// handleShareApprove — заказчик принял смету.
func (s *Server) handleShareApprove(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenOK(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	// атомарно: draft|sent → approved; возвратит владельца и название
	chatID, estID, title, err := s.svc.ApproveEstimateByToken(ctx, token)
	if err == store.ErrNotFound {
		writeErr(w, http.StatusNotFound, "Смета не найдена или уже согласована")
		return
	}
	if err != nil {
		apiErr(w, err, "смета")
		return
	}
	if _, err := s.svc.AddComment(ctx, estID, "client", "✅ Заказчик принял смету"); err != nil {
		log.Printf("webapp: approve comment: %v", err)
	}
	s.notify(func(n Notifier) { n.NotifyEstimateApproved(chatID, title) })
	writeJSON(w, http.StatusOK, map[string]any{"status": "approved"})
}

// handleShareComment — комментарий заказчика.
func (s *Server) handleShareComment(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenOK(w, r)
	if !ok {
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Text = strings.TrimSpace(in.Text)
	if len([]rune(in.Text)) < 1 || len([]rune(in.Text)) > 500 {
		writeErr(w, http.StatusBadRequest, "Комментарий — до 500 символов")
		return
	}
	chatID, estID, title, err := s.svc.ShareChatID(r.Context(), token)
	if err != nil {
		writeErr(w, http.StatusNotFound, "Смета не найдена")
		return
	}
	c, err := s.svc.AddComment(r.Context(), estID, "client", in.Text)
	if err != nil {
		apiErr(w, err, "комментарий")
		return
	}
	s.notify(func(n Notifier) { n.NotifyClientComment(chatID, title, in.Text) })
	writeJSON(w, http.StatusCreated, c)
}

// handleSharePrint — печатная версия страницы: PDF решается средствами
// браузера («Сохранить как PDF») — нулевая зависимость вместо gofpdf
// + шрифтов в бинарнике. TODO(v0.7): серверный PDF, если понадобится
// авто-вложение в письмо.
func (s *Server) handleSharePrint(w http.ResponseWriter, r *http.Request) {
	token, ok := tokenOK(w, r)
	if !ok {
		return
	}
	v, err := s.svc.GetShareView(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
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
	if total > 0.01 {
		v.HiddenShare = int(hidden/total*100 + 0.5)
	}
	v.CanApprove = false
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = shareTmpl.Execute(w, &v)
}
