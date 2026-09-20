// Кабинет заказчика — read-only endpoints + привязка (пилот 360).
package httpapi

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"proakt/internal/domain"
)

func clientRoutes(mux *http.ServeMux, s *Server) {
	mux.HandleFunc("GET /api/client/overview", s.auth(s.handleClientOverview))
	mux.HandleFunc("GET /api/client/work", s.auth(s.handleClientWork))
	mux.HandleFunc("GET /api/client/finance", s.auth(s.handleClientFinance))
	mux.HandleFunc("GET /api/client/docs", s.auth(s.handleClientDocs))
	mux.HandleFunc("GET /api/client/photos/{id}/file", s.auth(s.handleClientPhotoFile))
	mux.HandleFunc("GET /api/clients", s.auth(s.handleClientsList))
	mux.HandleFunc("POST /api/clients/grant", s.auth(s.handleClientGrant))
	mux.HandleFunc("POST /api/clients/revoke", s.auth(s.handleClientRevoke))
}

func clientObjectID(r *http.Request) (int64, bool) {
	v := r.URL.Query().Get("object_id")
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// mustClient — проверка доступа заказчика, чужое даёт 404.
func (s *Server) mustClient(w http.ResponseWriter, r *http.Request) (tg, oid int64, ok bool) {
	tg = chatOf(r)
	oid, ok = clientObjectID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "нет object_id")
		return 0, 0, false
	}
	allowed, err := s.svc.CanClientSee(r.Context(), tg, oid)
	if err != nil || !allowed {
		writeErr(w, http.StatusNotFound, "не найдено")
		return 0, 0, false
	}
	return tg, oid, true
}

// mustOwner — объект принадлежит вызывающему мастеру.
func (s *Server) mustOwner(w http.ResponseWriter, r *http.Request, oid int64) bool {
	obj, err := s.svc.GetObject(r.Context(), oid)
	if err != nil {
		writeErr(w, http.StatusNotFound, "не найдено")
		return false
	}
	if obj.ChatID != chatOf(r) {
		writeErr(w, http.StatusNotFound, "не найдено")
		return false
	}
	return true
}

func (s *Server) handleClientOverview(w http.ResponseWriter, r *http.Request) {
	_, oid, ok := s.mustClient(w, r)
	if !ok {
		return
	}
	objs, err := s.svc.ClientObjects(r.Context(), chatOf(r))
	if err != nil {
		apiErr(w, err, "объект")
		return
	}
	for _, o := range objs {
		if o.ID == oid {
			writeJSON(w, 200, map[string]any{"object": o, "progress": 91})
			return
		}
	}
	writeErr(w, 404, "не найдено")
}

func (s *Server) handleClientWork(w http.ResponseWriter, r *http.Request) {
	_, oid, ok := s.mustClient(w, r)
	if !ok {
		return
	}
	photos, err := s.svc.ListPhotos(r.Context(), oid, 0)
	if err != nil {
		apiErr(w, err, "фото")
		return
	}
	if photos == nil {
		photos = []domain.PhotoRec{}
	}
	writeJSON(w, 200, map[string]any{
		"object_id": oid,
		"photos":    photos,
		"report":    "assets/hidden-work-primer-report.webp",
	})
}

func (s *Server) handleClientFinance(w http.ResponseWriter, r *http.Request) {
	tg, oid, ok := s.mustClient(w, r)
	if !ok {
		return
	}
	_ = tg
	obj, err := s.svc.GetObject(r.Context(), oid)
	if err != nil {
		apiErr(w, err, "объект")
		return
	}
	acts, err := s.svc.ListActs(r.Context(), obj.ChatID, 200)
	if err != nil {
		apiErr(w, err, "акты")
		return
	}
	var mine []map[string]any
	var total, paid float64
	for _, a := range acts {
		if a.ObjectID != oid {
			continue
		}
		total += a.Total
		paid += a.Paid
		mine = append(mine, map[string]any{
			"id": a.ID, "act_no": a.ActNo, "total": a.Total,
			"paid": a.Paid, "date": a.Date, "photos": a.Photos,
		})
	}
	if mine == nil {
		mine = []map[string]any{}
	}
	writeJSON(w, 200, map[string]any{
		"object_id": oid, "acts": mine,
		"total": total, "paid": paid, "debt": total - paid,
		"demo_note": "оплаты 505000 и остаток — демонстрационные, Виталий правит их в админке",
	})
}

func (s *Server) handleClientDocs(w http.ResponseWriter, r *http.Request) {
	_, oid, ok := s.mustClient(w, r)
	if !ok {
		return
	}
	obj, err := s.svc.GetObject(r.Context(), oid)
	if err != nil {
		apiErr(w, err, "объект")
		return
	}
	acts, err := s.svc.ListActs(r.Context(), obj.ChatID, 200)
	if err != nil {
		apiErr(w, err, "акты")
		return
	}
	var docs []map[string]any
	for _, a := range acts {
		if a.ObjectID != oid {
			continue
		}
		docs = append(docs, map[string]any{
			"act_id": a.ID, "act_no": a.ActNo,
			"xlsx_url": "/api/acts/" + strconv.FormatInt(a.ID, 10) + "/xlsx",
		})
	}
	if docs == nil {
		docs = []map[string]any{}
	}
	writeJSON(w, 200, map[string]any{"object_id": oid, "docs": docs})
}

func (s *Server) handleClientsList(w http.ResponseWriter, r *http.Request) {
	oid, ok := clientObjectID(r)
	if !ok {
		writeErr(w, 400, "нет object_id")
		return
	}
	if !s.mustOwner(w, r, oid) {
		return
	}
	ids, err := s.svc.ClientList(r.Context(), oid)
	if err != nil {
		apiErr(w, err, "доступ")
		return
	}
	if ids == nil {
		ids = []int64{}
	}
	writeJSON(w, 200, map[string]any{"object_id": oid, "clients": ids})
}

func (s *Server) handleClientGrant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ObjectID int64 `json:"object_id"`
		TgUserID int64 `json:"tg_user_id"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.ObjectID <= 0 || req.TgUserID <= 0 {
		writeErr(w, 400, "нужны object_id и tg_user_id")
		return
	}
	if !s.mustOwner(w, r, req.ObjectID) {
		return
	}
	if err := s.svc.GrantClient(r.Context(), req.ObjectID, req.TgUserID); err != nil {
		apiErr(w, err, "привязка")
		return
	}
	writeJSON(w, 200, map[string]any{"granted": true})
}

func (s *Server) handleClientRevoke(w http.ResponseWriter, r *http.Request) {	var req struct {
		ObjectID int64 `json:"object_id"`
		TgUserID int64 `json:"tg_user_id"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.ObjectID <= 0 || req.TgUserID <= 0 {
		writeErr(w, 400, "нужны object_id и tg_user_id")
		return
	}
	if !s.mustOwner(w, r, req.ObjectID) {
		return
	}
	if err := s.svc.RevokeClient(r.Context(), req.ObjectID, req.TgUserID); err != nil {
		apiErr(w, err, "отвязка")
		return
	}
	writeJSON(w, 200, map[string]any{"revoked": true})
}

// servePhotoFile — отдать снимок с диска, путь обязан лежать в FILES_DIR.
func servePhotoFile(s *Server, w http.ResponseWriter, r *http.Request, filePath string) {
	if filePath == "" {
		writeErr(w, http.StatusNotFound, "Файл не сохранён на диске — фото живёт в Telegram (бот пришлёт по кнопке)")
		return
	}
	root, _ := filepath.Abs(s.cfg.FilesDir)
	full, err := filepath.Abs(filePath)
	if err != nil || !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		writeErr(w, http.StatusNotFound, "файл недоступен")
		return
	}
	f, err := os.Open(full)
	if err != nil {
		writeErr(w, http.StatusNotFound, "файл недоступен")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		writeErr(w, http.StatusNotFound, "файл недоступен")
		return
	}
	ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(full)))
	if ct == "" {
		ct = "image/jpeg"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, filepath.Base(full), st.ModTime(), f)
}

// handleClientPhotoFile — снимок заказчику: только свой объект, иначе 404.
func (s *Server) handleClientPhotoFile(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id фото")
		return
	}
	p, err := s.svc.ClientPhoto(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "фото")
		return
	}
	servePhotoFile(s, w, r, p.FilePath)
}
