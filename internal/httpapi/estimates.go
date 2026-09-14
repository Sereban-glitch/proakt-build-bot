// estimates.go — REST-обработчики смет, шаблонов и парсера строк (v0.5).
//
// Принципы те же, что в v0.4: чат-скоуп на каждый запрос (IDOR исключён —
// все методы store принимают chatID), валидация ввода до записи, ошибки
// бизнеса — 4xx, ошибки базы — 5xx.
package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"proakt/internal/domain"
	"proakt/internal/parse"
	"proakt/internal/xlsx"
)

// estOwned — проверка владения сметой (аналог actOwned).
func (s *Server) estOwned(w http.ResponseWriter, r *http.Request, estID int64) bool {
	chatID, err := s.svc.EstimateChatID(r.Context(), estID)
	if err != nil {
		apiErr(w, err, "смета")
		return false
	}
	if chatID != chatOf(r) {
		writeErr(w, http.StatusNotFound, "смета не найдена")
		return false
	}
	return true
}

// newShareToken — криптостойкий токен публичной ссылки (16 байт = 32 hex).
func newShareToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand падает только при сломанной системе — но мы не молчим
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// --- сметы ---------------------------------------------------------------------

func (s *Server) handleEstimatesList(w http.ResponseWriter, r *http.Request) {
	objID64 := int64(0)
	if v := r.URL.Query().Get("object_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			objID64 = n
		}
	}
	list, err := s.svc.ListEstimates(r.Context(), chatOf(r), objID64)
	if err != nil {
		apiErr(w, err, "сметы")
		return
	}
	if list == nil {
		list = []domain.EstimateBrief{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleEstimateCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ObjectID int64   `json:"object_id"`
		Title    string  `json:"title"`
		Coeff    float64 `json:"coeff"`
		Note     string  `json:"note"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if utf8.RuneCountInString(in.Title) < 2 || utf8.RuneCountInString(in.Title) > 80 {
		writeErr(w, http.StatusBadRequest, "Название сметы — от 2 до 80 символов")
		return
	}
	if in.ObjectID <= 0 {
		writeErr(w, http.StatusBadRequest, "Нужен object_id объекта")
		return
	}
	if in.Coeff == 0 {
		in.Coeff = 1
	}
	if in.Coeff < 1 || in.Coeff > 3 {
		writeErr(w, http.StatusBadRequest, "Коэффициент сложности — от 1.0 до 3.0")
		return
	}
	e, err := s.svc.CreateEstimate(r.Context(), chatOf(r), in.ObjectID, in.Title, moneyF(in.Coeff), strings.TrimSpace(in.Note))
	if err != nil {
		apiErr(w, err, "смета")
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// handleEstimateGet — смета + позиции (одним запросом, как handleActGet).
func (s *Server) handleEstimateGet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	ctx := r.Context()
	if !s.estOwned(w, r, id) {
		return
	}
	brief, err := s.svc.GetEstimateBrief(ctx, chatOf(r), id)
	if err != nil {
		apiErr(w, err, "смета")
		return
	}
	lines, err := s.svc.EstimateLines(ctx, id)
	if err != nil {
		apiErr(w, err, "позиции")
		return
	}
	if lines == nil {
		lines = []domain.EstimateLine{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"estimate": brief, "lines": lines})
}

// handleEstimatePatch — title/status/coeff/note (nil = не трогаем).
func (s *Server) handleEstimatePatch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	var in struct {
		Title  *string  `json:"title"`
		Status *string  `json:"status"`
		Coeff  *float64 `json:"coeff"`
		Note   *string  `json:"note"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if utf8.RuneCountInString(t) > 80 {
			writeErr(w, http.StatusBadRequest, "Название сметы — до 80 символов")
			return
		}
		in.Title = &t
	}
	if in.Coeff != nil && (*in.Coeff < 1 || *in.Coeff > 3) {
		writeErr(w, http.StatusBadRequest, "Коэффициент сложности — от 1.0 до 3.0")
		return
	}
	if !s.estOwned(w, r, id) {
		return
	}
	e, err := s.svc.UpdateEstimate(r.Context(), chatOf(r), id, in.Title, in.Status, in.Note, in.Coeff)
	if err != nil {
		apiErr(w, err, "смета")
		return
	}
	_ = e
	brief, err := s.svc.GetEstimateBrief(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "смета")
		return
	}
	writeJSON(w, http.StatusOK, brief)
}

func (s *Server) handleEstimateDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	if !s.estOwned(w, r, id) {
		return
	}
	brief, err := s.svc.DeleteEstimate(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "смета")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "estimate": brief})
}

// --- строки сметы ---------------------------------------------------------------

// handleEstLineAdd — одна позиция ИЛИ блок позиций (шаблон/мультистрока):
// {name,qty,unit,price,hidden,note} или {lines:[{name,qty,unit,price,hidden,note}]}.
func (s *Server) handleEstLineAdd(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	var in struct {
		Name   string  `json:"name"`
		Qty    float64 `json:"qty"`
		Unit   string  `json:"unit"`
		Price  float64 `json:"price"`
		Hidden bool    `json:"hidden"`
		Note   string  `json:"note"`
		Lines  []struct {
			Name   string  `json:"name"`
			Qty    float64 `json:"qty"`
			Unit   string  `json:"unit"`
			Price  float64 `json:"price"`
			Hidden bool    `json:"hidden"`
			Note   string  `json:"note"`
		} `json:"lines"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if !s.estOwned(w, r, id) {
		return
	}

	if len(in.Lines) > 0 {
		// bulk: шаблон или мультистрока
		if len(in.Lines) > 100 {
			writeErr(w, http.StatusBadRequest, "Максимум 100 позиций за раз")
			return
		}
		var bulk []domain.EstimateLine
		for _, li := range in.Lines {
			li.Name = strings.TrimSpace(li.Name)
			if utf8.RuneCountInString(li.Name) < 2 || utf8.RuneCountInString(li.Name) > 120 {
				writeErr(w, http.StatusBadRequest, "В названии позиции должно быть от 2 до 120 символов: "+li.Name)
				return
			}
			if li.Price < 0 || li.Price > 1e9 || li.Qty < 0 || li.Qty > 1e7 {
				writeErr(w, http.StatusBadRequest, "Количество/цена вне допустимого диапазона")
				return
			}
			bulk = append(bulk, domain.EstimateLine{
				Name: li.Name, Qty: moneyF(li.Qty), Unit: parse.NormalizeUnit(li.Unit),
				Price: moneyF(li.Price), Hidden: li.Hidden, Note: strings.TrimSpace(li.Note),
			})
		}
		n, err := s.svc.AddEstimateLinesBulk(r.Context(), chatOf(r), id, bulk)
		if err != nil {
			apiErr(w, err, "позиции")
			return
		}
		brief, _ := s.svc.GetEstimateBrief(r.Context(), chatOf(r), id)
		writeJSON(w, http.StatusCreated, map[string]any{"added": n, "estimate": brief})
		return
	}

	// одна позиция
	in.Name = strings.TrimSpace(in.Name)
	if utf8.RuneCountInString(in.Name) < 2 || utf8.RuneCountInString(in.Name) > 120 {
		writeErr(w, http.StatusBadRequest, "Название позиции — от 2 до 120 символов")
		return
	}
	if in.Price < 0 || in.Price > 1e9 || in.Qty < 0 || in.Qty > 1e7 {
		writeErr(w, http.StatusBadRequest, "Количество/цена вне допустимого диапазона")
		return
	}
	l, err := s.svc.AddEstimateLine(r.Context(), chatOf(r), id, in.Name, parse.NormalizeUnit(in.Unit), moneyF(in.Qty), moneyF(in.Price), in.Hidden, strings.TrimSpace(in.Note))
	if err != nil {
		apiErr(w, err, "позиция")
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

// handleEstLinePatch — qty/price/name/unit/hidden/done/note.
func (s *Server) handleEstLinePatch(w http.ResponseWriter, r *http.Request) {
	lineID, ok := pathID(r, "lid")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id позиции")
		return
	}
	var in struct {
		Name   *string  `json:"name"`
		Unit   *string  `json:"unit"`
		Note   *string  `json:"note"`
		Qty    *float64 `json:"qty"`
		Price  *float64 `json:"price"`
		Hidden *bool    `json:"hidden"`
		Done   *bool    `json:"done"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Name != nil {
		*in.Name = strings.TrimSpace(*in.Name)
		if utf8.RuneCountInString(*in.Name) < 2 || utf8.RuneCountInString(*in.Name) > 120 {
			writeErr(w, http.StatusBadRequest, "Название позиции — от 2 до 120 символов")
			return
		}
	}
	if in.Price != nil && (*in.Price < 0 || *in.Price > 1e9) {
		writeErr(w, http.StatusBadRequest, "Цена вне допустимого диапазона")
		return
	}
	if in.Qty != nil && (*in.Qty < 0 || *in.Qty > 1e7) {
		writeErr(w, http.StatusBadRequest, "Количество вне допустимого диапазона")
		return
	}
	// владение строкой проверяется в store (JOIN с estimates.chat_id)
	if in.Unit != nil {
		u := parse.NormalizeUnit(*in.Unit)
		in.Unit = &u
	}
	if in.Qty != nil {
		q := moneyF(*in.Qty)
		in.Qty = &q
	}
	if in.Price != nil {
		p := moneyF(*in.Price)
		in.Price = &p
	}
	l, err := s.svc.UpdateEstimateLine(r.Context(), chatOf(r), lineID, in.Name, in.Unit, in.Note, in.Qty, in.Price, in.Hidden, in.Done)
	if err != nil {
		apiErr(w, err, "позиция")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) handleEstLineDelete(w http.ResponseWriter, r *http.Request) {
	lineID, ok := pathID(r, "lid")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id позиции")
		return
	}
	l, err := s.svc.DeleteEstimateLine(r.Context(), chatOf(r), lineID)
	if err != nil {
		apiErr(w, err, "позиция")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "line": l})
}

func (s *Server) handleEstLineMove(w http.ResponseWriter, r *http.Request) {
	lineID, ok := pathID(r, "lid")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id позиции")
		return
	}
	var in struct {
		Dir string `json:"dir"` // up | down
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Dir != "up" && in.Dir != "down" {
		writeErr(w, http.StatusBadRequest, "dir должен быть up или down")
		return
	}
	if err := s.svc.MoveEstimateLine(r.Context(), chatOf(r), lineID, in.Dir == "up"); err != nil {
		writeErr(w, http.StatusBadRequest, "Двигать некуда — позиция у края списка")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"moved": true})
}

// --- share / xlsx ----------------------------------------------------------------

// handleEstimateShare — создать/перевыпустить ссылку заказчику.
func (s *Server) handleEstimateShare(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	if !s.estOwned(w, r, id) {
		return
	}
	token, err := s.svc.SetShareToken(r.Context(), chatOf(r), id, newShareToken())
	if err != nil {
		apiErr(w, err, "ссылка")
		return
	}
	url := token
	if s.cfg.PublicURL != "" {
		url = strings.TrimRight(s.cfg.PublicURL, "/") + "/s/" + token
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "url": url})
}

// handleEstimateXLSX — файл сметы в формате таблицы мастера.
func (s *Server) handleEstimateXLSX(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	ctx := r.Context()
	if !s.estOwned(w, r, id) {
		return
	}
	brief, err := s.svc.GetEstimateBrief(ctx, chatOf(r), id)
	if err != nil {
		apiErr(w, err, "смета")
		return
	}
	lines, err := s.svc.EstimateLines(ctx, id)
	if err != nil {
		apiErr(w, err, "позиции")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	name := xlsx.EstimateFileName(brief)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", sanitizeName(name)))
	w.Header().Set("Cache-Control", "no-store")
	if err := xlsx.WriteEstimate(w, brief, lines); err != nil {
		log.Printf("webapp: xlsx сметы %d: %v", id, err)
	}
}

// sanitizeName — латиница для filename=, кириллица уедет в RFC 5987-часть.
func sanitizeName(name string) string {
	repl := strings.NewReplacer(" ", "_", "\"", "", ";", "", ",", "", "\\", "", "/", "")
	out := repl.Replace(name)
	var b strings.Builder
	for _, ch := range out {
		if ch > 32 && ch < 127 {
			b.WriteRune(ch)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// --- парсер строки (та же логика, что у бота в чате) ------------------------------

// handleParseLine — «шлифовка 45м 50, грунтовка 45» → позиции для предпросмотра.
// Фронт показывает распознанное ДО добавления: деньги святы (философия v0.3.6).
func (s *Server) handleParseLine(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Text = strings.TrimSpace(in.Text)
	if in.Text == "" || len(in.Text) > 500 {
		writeErr(w, http.StatusBadRequest, "Строка пуста или слишком длинная")
		return
	}
	parts := parse.SplitPositions(in.Text)
	if len(parts) == 0 {
		// одиночная позиция: SplitPositions режет только настоящие мультистроки
		parts = []string{in.Text}
	}
	if parse.SuspiciousMulti(in.Text) {
		writeJSON(w, http.StatusOK, map[string]any{
			"suspicious": true,
			"lines":      []domain.DraftLine{},
			"hint":       "Похоже, в одной строке несколько позиций без цен — добавь каждую отдельно: одна строка = одна позиция",
		})
		return
	}
	out := make([]domain.DraftLine, 0, len(parts))
	for _, p := range parts {
		if dl, ok := parse.ParsePosition(p); ok {
			out = append(out, dl)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"suspicious": false, "lines": out})
}

// --- шаблоны работ ----------------------------------------------------------------

func (s *Server) handleTemplatesList(w http.ResponseWriter, r *http.Request) {
	list, err := s.svc.ListTemplates(r.Context(), chatOf(r))
	if err != nil {
		apiErr(w, err, "шаблоны")
		return
	}
	if list == nil {
		list = []domain.Template{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list, "count": len(list)})
}

// handleTemplateUpsert — {name, lines:[{name,qty,unit,price,hidden}]}.
func (s *Server) handleTemplateUpsert(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string                `json:"name"`
		Lines []domain.TemplateLine `json:"lines"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if utf8.RuneCountInString(in.Name) < 2 || utf8.RuneCountInString(in.Name) > 60 {
		writeErr(w, http.StatusBadRequest, "Название шаблона — от 2 до 60 символов")
		return
	}
	if len(in.Lines) == 0 || len(in.Lines) > 60 {
		writeErr(w, http.StatusBadRequest, "В шаблоне должно быть от 1 до 60 позиций")
		return
	}
	for i := range in.Lines {
		in.Lines[i].Name = strings.TrimSpace(in.Lines[i].Name)
		if utf8.RuneCountInString(in.Lines[i].Name) < 2 {
			writeErr(w, http.StatusBadRequest, "В названии позиции шаблона должно быть от 2 символов")
			return
		}
		if in.Lines[i].Price < 0 || in.Lines[i].Qty < 0 {
			writeErr(w, http.StatusBadRequest, "Количество/цена не могут быть отрицательными")
			return
		}
	}
	t, err := s.svc.UpsertTemplate(r.Context(), chatOf(r), in.Name, in.Lines)
	if err != nil {
		apiErr(w, err, "шаблон")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleTemplateDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id шаблона")
		return
	}
	t, ok, err := s.svc.DeleteTemplate(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "шаблон")
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "Шаблон не найден")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "template": t})
}
