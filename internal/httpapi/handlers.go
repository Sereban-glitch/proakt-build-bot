package httpapi

import (
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"proakt/internal/domain"
	"proakt/internal/parse"
)

// routes — таблица маршрутов (Go 1.22 pattern routing).
func routes(mux *http.ServeMux, s *Server) {
	a := s.auth
	mux.HandleFunc("GET /api/dashboard", a(s.handleDashboard))
	mux.HandleFunc("GET /api/objects", a(s.handleObjectsList))
	mux.HandleFunc("POST /api/objects", a(s.handleObjectCreate))
	mux.HandleFunc("POST /api/objects/{id}/archive", a(s.handleObjectArchive))
	mux.HandleFunc("GET /api/acts", a(s.handleActsList))
	mux.HandleFunc("GET /api/acts/{id}", a(s.handleActGet))
	mux.HandleFunc("DELETE /api/acts/{id}", a(s.handleActDelete))
	mux.HandleFunc("POST /api/acts/{id}/payments", a(s.handlePaymentCreate))
	mux.HandleFunc("DELETE /api/payments/{id}", a(s.handlePaymentDelete))
	mux.HandleFunc("GET /api/price", a(s.handlePriceList))
	mux.HandleFunc("POST /api/price", a(s.handlePriceUpsert))
	mux.HandleFunc("DELETE /api/price/{name}", a(s.handlePriceDelete))
	mux.HandleFunc("GET /api/photos", a(s.handlePhotosList))
	mux.HandleFunc("GET /api/photos/{id}/file", a(s.handlePhotoFile))
	mux.HandleFunc("DELETE /api/photos/{id}", a(s.handlePhotoDelete))

	// --- сметы (v0.5) ---
	mux.HandleFunc("GET /api/estimates", a(s.handleEstimatesList))
	mux.HandleFunc("POST /api/estimates", a(s.handleEstimateCreate))
	mux.HandleFunc("GET /api/estimates/{id}", a(s.handleEstimateGet))
	mux.HandleFunc("PATCH /api/estimates/{id}", a(s.handleEstimatePatch))
	mux.HandleFunc("DELETE /api/estimates/{id}", a(s.handleEstimateDelete))
	mux.HandleFunc("POST /api/estimates/{id}/lines", a(s.handleEstLineAdd))
	mux.HandleFunc("PATCH /api/estimates/{id}/lines/{lid}", a(s.handleEstLinePatch))
	mux.HandleFunc("DELETE /api/estimates/{id}/lines/{lid}", a(s.handleEstLineDelete))
	mux.HandleFunc("POST /api/estimates/{id}/lines/{lid}/move", a(s.handleEstLineMove))
	mux.HandleFunc("POST /api/estimates/{id}/share", a(s.handleEstimateShare))
	mux.HandleFunc("GET /api/estimates/{id}/xlsx", a(s.handleEstimateXLSX))

	// парсер строки бота (единая логика с FSM): «шпаклёвка 45 м² 140»
	mux.HandleFunc("POST /api/parse-line", a(s.handleParseLine))

	// шаблоны работ («админка», v0.5)
	mux.HandleFunc("GET /api/templates", a(s.handleTemplatesList))
	mux.HandleFunc("POST /api/templates", a(s.handleTemplateUpsert))
	mux.HandleFunc("DELETE /api/templates/{id}", a(s.handleTemplateDelete))

	// --- v0.6: герой-сценарий, диалог, Google Таблица ---
	mux.HandleFunc("POST /api/estimates/{id}/voice", a(s.handleEstimateVoice))
	mux.HandleFunc("POST /api/estimates/{id}/act", a(s.handleEstimateAct))
	mux.HandleFunc("GET /api/estimates/{id}/comments", a(s.handleCommentsList))
	mux.HandleFunc("POST /api/estimates/{id}/comments", a(s.handleCommentAdd))
	mux.HandleFunc("POST /api/estimates/{id}/lines/{lid}/photo", a(s.handleEstLinePhoto))
	mux.HandleFunc("GET /api/price/suggest", a(s.handlePriceSuggest))
	mux.HandleFunc("GET /api/rooms/presets", a(s.handleRoomPresets))
	mux.HandleFunc("POST /api/rooms/apply", a(s.handleRoomApply))
	mux.HandleFunc("GET /api/price/source", a(s.handlePriceSource))
	mux.HandleFunc("POST /api/price/sync", a(s.handlePriceSync))

	// публичный просмотр сметы заказчиком (без auth, доступ по токену)
	mux.HandleFunc("GET /s/{token}", s.handleSharePage)
	mux.HandleFunc("GET /s/{token}/photo/{id}", s.handleSharePhoto)
	mux.HandleFunc("POST /s/{token}/approve", s.handleShareApprove)
	mux.HandleFunc("POST /s/{token}/comment", s.handleShareComment)
	mux.HandleFunc("GET /s/{token}/print", s.handleSharePrint)
}

// --- дашборд -----------------------------------------------------------------

// handleDashboard — всё для главного экрана: сводка, долги, последние акты.
// Логика «упущенной выгоды» повторяет showReport бота (деньги мимо кармана).
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatID := chatOf(r)

	st, err := s.svc.Stats(ctx, chatID)
	if err != nil {
		apiErr(w, err, "статистика")
		return
	}
	priceN, _ := s.svc.CatalogCount(ctx, chatID)

	acts, err := s.svc.ListActs(ctx, chatID, 50)
	if err != nil {
		apiErr(w, err, "акты")
		return
	}
	recent := acts
	if len(recent) > 5 {
		recent = recent[:5]
	}
	var debts []domain.ActBrief
	for _, a := range acts {
		if a.Balance() > 0.009 {
			debts = append(debts, a)
		}
		if len(debts) >= 10 {
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":        map[string]any{"chat_id": chatID},
		"stats":       st,
		"price_count": priceN,
		"debt_total":  moneyF(st.Total - st.Paid),
		"debts":       debts,
		"recent_acts": recent,
	})
}

// --- объекты -----------------------------------------------------------------

func (s *Server) handleObjectsList(w http.ResponseWriter, r *http.Request) {
	objs, err := s.svc.ListObjects(r.Context(), chatOf(r))
	if err != nil {
		apiErr(w, err, "объекты")
		return
	}
	if objs == nil {
		objs = []domain.ObjectBrief{}
	}
	writeJSON(w, http.StatusOK, objs)
}

func (s *Server) handleObjectCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Customer string `json:"customer"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if utf8.RuneCountInString(in.Name) < 2 || utf8.RuneCountInString(in.Name) > 60 {
		writeErr(w, http.StatusBadRequest, "Название должно быть от 2 до 60 символов")
		return
	}
	o, err := s.svc.CreateObject(r.Context(), chatOf(r), in.Name, strings.TrimSpace(in.Customer))
	if err != nil {
		apiErr(w, err, "объект")
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) handleObjectArchive(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id объекта")
		return
	}
	name, err := s.svc.ArchiveObject(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "объект")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"archived": true, "name": name})
}

// --- акты --------------------------------------------------------------------

func (s *Server) handleActsList(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	acts, err := s.svc.ListActs(r.Context(), chatOf(r), limit)
	if err != nil {
		apiErr(w, err, "акты")
		return
	}
	if acts == nil {
		acts = []domain.ActBrief{}
	}
	writeJSON(w, http.StatusOK, acts)
}

// handleActGet — акт + позиции + оплаты + фото (chat-скоуп проверен в SQL).
func (s *Server) handleActGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id акта")
		return
	}
	ctx := r.Context()
	if !s.actOwned(w, r, id) {
		return
	}
	brief, err := s.svc.GetAct(ctx, id)
	if err != nil {
		apiErr(w, err, "акт")
		return
	}
	lines, err := s.svc.ActLines(ctx, id)
	if err != nil {
		apiErr(w, err, "позиции")
		return
	}
	payments, err := s.svc.Payments(ctx, id)
	if err != nil {
		apiErr(w, err, "оплаты")
		return
	}
	photos, err := s.svc.ListPhotos(ctx, brief.ObjectID, id)
	if err != nil {
		apiErr(w, err, "фото")
		return
	}
	if lines == nil {
		lines = []domain.ActLine{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"act":      brief,
		"lines":    lines,
		"payments": payments,
		"photos":   photos,
	})
}

func (s *Server) handleActDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id акта")
		return
	}
	if !s.actOwned(w, r, id) {
		return
	}
	brief, err := s.svc.DeleteAct(r.Context(), id)
	if err != nil {
		apiErr(w, err, "акт")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "act": brief})
}

// actOwned — DeleteAct/GetAct в store не скоупятся по чату (боту это не нужно:
// id приходят только из его же кнопок). API обязан проверять владение сам —
// иначе чужой actID из интернета удалит чужой акт (IDOR).
func (s *Server) actOwned(w http.ResponseWriter, r *http.Request, actID int64) bool {
	chatID, err := s.svc.ActChatID(r.Context(), actID)
	if err != nil {
		apiErr(w, err, "акт")
		return false
	}
	if chatID != chatOf(r) {
		log.Printf("webapp: чат %d лезет в акт %d (чат %d) — отказ", chatOf(r), actID, chatID)
		writeErr(w, http.StatusNotFound, "акт не найдено")
		return false
	}
	return true
}

// --- оплаты ------------------------------------------------------------------

func (s *Server) handlePaymentCreate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id акта")
		return
	}
	var in struct {
		Amount float64 `json:"amount"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Amount <= 0 || in.Amount > 1e9 {
		writeErr(w, http.StatusBadRequest, "Сумма должна быть положительным числом")
		return
	}
	if !s.actOwned(w, r, id) {
		return
	}
	if err := s.svc.CreatePayment(r.Context(), id, in.Amount, ""); err != nil {
		apiErr(w, err, "оплата")
		return
	}
	brief, err := s.svc.GetAct(r.Context(), id)
	if err != nil {
		apiErr(w, err, "акт")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"act": brief})
}

func (s *Server) handlePaymentDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id оплаты")
		return
	}
	rec, err := s.svc.DeletePayment(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "оплата")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "payment": rec})
}

// --- прайс -------------------------------------------------------------------

func (s *Server) handlePriceList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatID := chatOf(r)
	items, err := s.svc.ListCatalog(ctx, chatID)
	if err != nil {
		apiErr(w, err, "прайс")
		return
	}
	if items == nil {
		items = []domain.CatalogItem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

// handlePriceUpsert — добавить/обновить цену. Имя нормализуем тем же
// parse.NormName, что и бот: «Штукатурка» и «штукатурка» — одна позиция.
func (s *Server) handlePriceUpsert(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string  `json:"name"`
		Unit  string  `json:"unit"`
		Price float64 `json:"price"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	name := parse.NormName(in.Name)
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 80 {
		writeErr(w, http.StatusBadRequest, "Название позиции — от 2 до 80 символов")
		return
	}
	if in.Price <= 0 || in.Price > 1e9 {
		writeErr(w, http.StatusBadRequest, "Цена должна быть положительным числом")
		return
	}
	prev, existed, err := s.svc.UpsertCatalogItem(r.Context(), chatOf(r), name, parse.NormalizeUnit(in.Unit), in.Price)
	if err != nil {
		apiErr(w, err, "прайс")
		return
	}
	n, _ := s.svc.CatalogCount(r.Context(), chatOf(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"item":    domain.CatalogItem{Name: name, Unit: parse.NormalizeUnit(in.Unit), Price: in.Price},
		"prev":    moneyF(prev),
		"existed": existed,
		"changed": existed && prev != in.Price,
		"count":   n,
	})
}

func (s *Server) handlePriceDelete(w http.ResponseWriter, r *http.Request) {
	name := parse.NormName(r.PathValue("name"))
	item, ok, err := s.svc.DeleteCatalogItem(r.Context(), chatOf(r), name)
	if err != nil {
		apiErr(w, err, "прайс")
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "Позиции с таким названием нет в прайсе")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "item": item})
}

// --- фото --------------------------------------------------------------------

func (s *Server) handlePhotosList(w http.ResponseWriter, r *http.Request) {
	objID, _ := pathID(r, "object_id")
	if objID <= 0 {
		writeErr(w, http.StatusBadRequest, "Нужен object_id")
		return
	}
	ctx := r.Context()
	// скоуп: объект должен принадлежать чату
	o, err := s.svc.GetObject(ctx, objID)
	if err != nil || o.ChatID != chatOf(r) {
		writeErr(w, http.StatusNotFound, "объект не найдено")
		return
	}
	photos, err := s.svc.ListPhotos(ctx, objID, 0)
	if err != nil {
		apiErr(w, err, "фото")
		return
	}
	if photos == nil {
		photos = []domain.PhotoRec{}
	}
	writeJSON(w, http.StatusOK, photos)
}

// handlePhotoFile — отдаём снимок с диска (FILES_DIR), только владельцу.
// В Telegram бот показывает фото по file_id; в Mini App удобнее прямая ссылка.
func (s *Server) handlePhotoFile(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id фото")
		return
	}
	p, err := s.svc.GetPhoto(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "фото")
		return
	}
	if p.FilePath == "" {
		writeErr(w, http.StatusNotFound, "Файл не сохранён на диске — фото живёт в Telegram (бот пришлёт по кнопке)")
		return
	}
	// защита от path traversal: путь обязан лежать внутри FILES_DIR
	root, _ := filepath.Abs(s.cfg.FilesDir)
	full, err := filepath.Abs(p.FilePath)
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

func (s *Server) handlePhotoDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id фото")
		return
	}
	rec, err := s.svc.DeletePhoto(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "фото")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "photo": rec})
}

// atoi — как strconv.Atoi, но возвращает только то, что нужно маршруту.
func atoi(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, errors.New("пусто")
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("не цифра: %q", ch)
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}
