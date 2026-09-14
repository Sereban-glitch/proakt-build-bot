// v06.go — REST-обработчики v0.6:
//
//	POST /api/estimates/{id}/voice            — голос → позиции (мини-апп на объекте)
//	GET  /api/price/suggest?q=…               — подсказки из прайса («меньше печатать»)
//	POST /api/estimates/{id}/act              — акт из сметы за галочку
//	GET/POST /api/estimates/{id}/comments     — диалог мастера и заказчика
//	POST /api/estimates/{id}/lines/{lid}/photo — фото строки (multipart)
//	GET  /api/rooms/presets                   — типовые помещения с нормами
//	POST /api/rooms/apply                     — вставка помещения по размерам
//	POST /api/price/sync                      — синк с Google Таблицей сейчас
//	GET  /api/price/source                    — состояние источника прайса
package httpapi

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"proakt/internal/domain"
	"proakt/internal/gsheet"
	"proakt/internal/parse"
	"proakt/internal/rooms"
	"proakt/internal/store"
)

// --- голос в мини-апп (герой-сценарий: диктовка на объекте) ----------------------

// handleEstimateVoice — {audio: base64, mime: "audio/ogg"} → позиции.
// Тот же ai.Gateway, что у бота: транскрипт → JSON-позиции → цены из прайса
// подставит фронт (bulk-добавление с ценой). Голос нигде не хранится: OGG
// живёт в памяти до конца запроса.
func (s *Server) handleEstimateVoice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	if s.ai == nil {
		writeErr(w, http.StatusServiceUnavailable, "Голос не подключён на сервере (AI_GATEWAY_URL пуст) — вводи строкой")
		return
	}
	if !s.estOwned(w, r, id) {
		return
	}
	var in struct {
		Audio string `json:"audio"`
		Mime  string `json:"mime"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.Audio == "" || len(in.Audio) > 12<<20 { // ~9 МБ аудио в base64
		writeErr(w, http.StatusBadRequest, "Аудио пустое или слишком большое (лимит ~9 МБ)")
		return
	}
	ogg, err := base64.StdEncoding.DecodeString(in.Audio)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Аудио не в base64")
		return
	}
	if len(ogg) > 10<<20 {
		writeErr(w, http.StatusBadRequest, "Аудио длиннее 10 МБ — диктуй порциями")
		return
	}
	if in.Mime == "" {
		in.Mime = "audio/ogg"
	}

	ctx := r.Context()
	transcript, err := s.ai.TranscribeMime(ctx, ogg, in.Mime)
	if err != nil {
		log.Printf("webapp: голос сметы %d: %v", id, err)
		writeErr(w, http.StatusBadGateway, "Не получилось расшифровать — попробуй ещё раз или введи строкой")
		return
	}
	lines, err := s.ai.ExtractPositions(ctx, transcript)
	if err != nil || len(lines) == 0 {
		// запасной путь — локальный парсер (как у бота)
		lines = nil
		for _, ln := range strings.Split(transcript, "\n") {
			if dl, ok := parse.ParsePosition(strings.TrimSpace(ln)); ok {
				lines = append(lines, dl)
			}
		}
		if len(lines) == 0 {
			writeErr(w, http.StatusUnprocessableEntity, "Не выделил позиции — продиктуй медленнее: «кухня, стены, штукатурка, сорок квадратов»")
			return
		}
	}
	for i := range lines {
		if u := parse.NormalizeUnit(lines[i].Unit); u != "" {
			lines[i].Unit = u
		}
	}
	// подставляем цены из прайса сразу: мастер слышит сумму мгновенно
	items, _ := s.svc.ListCatalog(ctx, chatOf(r))
	var missing []string
	for i := range lines {
		if lines[i].Price > 0 {
			continue
		}
		if it, ok := matchCatalogItem(items, lines[i].Name); ok {
			lines[i].Price = it.Price
			lines[i].Sum = lines[i].Qty * it.Price
			if lines[i].Unit == "" {
				lines[i].Unit = it.Unit
			}
		} else {
			missing = append(missing, lines[i].Name)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"transcript": transcript,
		"lines":      lines,
		"missing":    missing,
	})
}

// matchCatalogItem — точное → вхождение (та же логика, что MatchCatalog бота).
func matchCatalogItem(items []domain.CatalogItem, query string) (domain.CatalogItem, bool) {
	q := parse.NormName(query)
	if q == "" {
		return domain.CatalogItem{}, false
	}
	for _, it := range items {
		if it.Name == q {
			return it, true
		}
	}
	if len([]rune(q)) >= 4 {
		for _, it := range items {
			if len([]rune(it.Name)) >= 3 && (strings.Contains(it.Name, q) || strings.Contains(q, it.Name)) {
				return it, true
			}
		}
	}
	return domain.CatalogItem{}, false
}

// --- подсказки из прайса -------------------------------------------------------

// handlePriceSuggest — топ-8 совпадений (для «умных подсказок» при диктовке).
func (s *Server) handlePriceSuggest(w http.ResponseWriter, r *http.Request) {
	q := parse.NormName(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{"items": []domain.CatalogItem{}})
		return
	}
	items, err := s.svc.ListCatalog(r.Context(), chatOf(r))
	if err != nil {
		apiErr(w, err, "прайс")
		return
	}
	out := make([]domain.CatalogItem, 0, 8)
	for _, it := range items {
		if it.Name == q || strings.Contains(it.Name, q) || strings.Contains(q, it.Name) {
			out = append(out, it)
			if len(out) >= 8 {
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// --- акт из сметы (еженедельная рутина за галочку) --------------------------------

// handleEstimateAct — {line_ids:[…]} или {all_pending:true} → акт + XLSX-ссылка.
func (s *Server) handleEstimateAct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	var in struct {
		LineIDs    []int64 `json:"line_ids"`
		AllPending bool    `json:"all_pending"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if !s.estOwned(w, r, id) {
		return
	}
	if len(in.LineIDs) > 100 {
		writeErr(w, http.StatusBadRequest, "Максимум 100 строк за акт")
		return
	}
	brief, n, err := s.svc.ActFromEstimate(r.Context(), chatOf(r), id, in.LineIDs)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusUnprocessableEntity, "Закрывать нечего: выбранные строки без суммы или уже закрыты")
		return
	}
	if err != nil {
		apiErr(w, err, "акт")
		return
	}
	lines, _ := s.svc.ActLines(r.Context(), brief.ID)
	writeJSON(w, http.StatusCreated, map[string]any{
		"act":       brief,
		"closed":    n,
		"act_lines": lines,
		"xlsx_url":  fmt.Sprintf("/api/acts/%d/xlsx", brief.ID),
	})
}

// --- комментарии к смете ------------------------------------------------------------

func (s *Server) handleCommentsList(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	if !s.estOwned(w, r, id) {
		return
	}
	list, err := s.svc.ListComments(r.Context(), chatOf(r), id)
	if err != nil {
		apiErr(w, err, "комментарии")
		return
	}
	if list == nil {
		list = []domain.EstimateComment{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *Server) handleCommentAdd(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
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
	if !s.estOwned(w, r, id) {
		return
	}
	c, err := s.svc.AddComment(r.Context(), id, "master", in.Text)
	if err != nil {
		apiErr(w, err, "комментарий")
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// --- фото строки сметы -----------------------------------------------------------

// handleEstLinePhoto — multipart (поле file) → сохранить на диск и привязать
// к строке. «Фотографируешь комнату на ходу — фото цепляется к последней
// продиктованной строке».
func (s *Server) handleEstLinePhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id сметы")
		return
	}
	lineID, ok := pathID(r, "lid")
	if !ok {
		writeErr(w, http.StatusBadRequest, "Некорректный id позиции")
		return
	}
	ctx := r.Context()
	if !s.estOwned(w, r, id) {
		return
	}
	l, err := s.svc.EstimateLineOwned(ctx, chatOf(r), lineID)
	if err != nil {
		apiErr(w, err, "позиция")
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "Файл не принят (лимит 10 МБ)")
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Нет файла в поле file")
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".heic" {
		writeErr(w, http.StatusBadRequest, "Жду фото (jpg/png/webp/heic)")
		return
	}
	caption := strings.TrimSpace(r.FormValue("caption"))
	// подпись по умолчанию — название строки: фото само объясняет себя
	if caption == "" {
		caption = l.Name
	}

	estID64 := id
	// файл: files/photos/obj_<id>/line_<lineID>_<ts>.<ext>
	svc := s.svc
	brief, err := svc.GetEstimateBrief(ctx, chatOf(r), estID64)
	if err != nil {
		apiErr(w, err, "смета")
		return
	}
	dir := filepath.Join(s.cfg.FilesDir, "photos", fmt.Sprintf("obj_%d", brief.ObjectID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		apiErr(w, err, "фото")
		return
	}
	path := filepath.Join(dir, fmt.Sprintf("line_%d_%d%s", lineID, time.Now().Unix(), ext))
	out, err := os.Create(path)
	if err != nil {
		apiErr(w, err, "фото")
		return
	}
	_, cpErr := io.Copy(out, io.LimitReader(file, 10<<20))
	_ = out.Close()
	if cpErr != nil {
		_ = os.Remove(path)
		writeErr(w, http.StatusInternalServerError, "Файл не сохранился")
		return
	}
	// file_id пуст: в Telegram это фото не уезжало — покажем в мини-апп и на share-странице по файлу
	if err := s.svc.AddEstLinePhoto(ctx, chatOf(r), id, lineID, "", path, caption); err != nil {
		_ = os.Remove(path)
		apiErr(w, err, "фото")
		return
	}
	photos, _ := s.svc.ListLinePhotos(ctx, chatOf(r), id, lineID)
	writeJSON(w, http.StatusCreated, map[string]any{"saved": true, "photos": photos})
}

// --- типовые помещения (нормы на бэкенде — единые для UI и бота) ---------------------

// handleRoomPresets — список пресетов с описанием норм.
func (s *Server) handleRoomPresets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": rooms.Presets})
}

// handleRoomApply — {est_id, preset, area, height, perimeter} → вставка блока.
// Цены подставляются из прайса, что найдётся; без цены — 0 (мастер заполнит).
func (s *Server) handleRoomApply(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EstID     int64   `json:"est_id"`
		Preset    string  `json:"preset"`
		Area      float64 `json:"area"`
		Height    float64 `json:"height"`
		Perimeter float64 `json:"perimeter"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if in.EstID <= 0 {
		writeErr(w, http.StatusBadRequest, "Нужен est_id")
		return
	}
	if in.Area <= 0 || in.Area > 10000 {
		writeErr(w, http.StatusBadRequest, "Площадь должна быть от 0 до 10000 м²")
		return
	}
	if in.Height <= 0 || in.Height > 20 {
		in.Height = 2.7
	}
	p, ok := rooms.PresetByKey(in.Preset)
	if !ok {
		writeErr(w, http.StatusBadRequest, "Неизвестный тип помещения")
		return
	}
	if !s.estOwned(w, r, in.EstID) {
		return
	}
	ctx := r.Context()
	built := rooms.Build(p, in.Area, in.Height, in.Perimeter)
	items, _ := s.svc.ListCatalog(ctx, chatOf(r))
	missing := 0
	for i := range built {
		if it, ok := matchCatalogItem(items, built[i].Name); ok {
			built[i].Price = it.Price
			built[i].Sum = built[i].Qty * it.Price
		} else {
			built[i].Sum = 0
			missing++
		}
	}
	n, err := s.svc.AddEstimateLinesBulk(ctx, chatOf(r), in.EstID, built)
	if err != nil {
		apiErr(w, err, "позиции")
		return
	}
	brief, _ := s.svc.GetEstimateBrief(ctx, chatOf(r), in.EstID)
	writeJSON(w, http.StatusCreated, map[string]any{
		"added": n, "missing_prices": missing, "estimate": brief, "lines": built,
	})
}

// --- Google Таблица: синк из мини-апп ------------------------------------------------

// handlePriceSource — текущее состояние источника прайса чата.
func (s *Server) handlePriceSource(w http.ResponseWriter, r *http.Request) {
	src, err := s.svc.PriceSource(r.Context(), chatOf(r))
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"connected": false})
		return
	}
	if err != nil {
		apiErr(w, err, "источник")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connected": true, "source": src})
}

// handlePriceSync — синк сейчас (мерж: обновить + добавить, ничего не удалить).
func (s *Server) handlePriceSync(w http.ResponseWriter, r *http.Request) {
	chatID := chatOf(r)
	src, err := s.svc.PriceSource(r.Context(), chatID)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusBadRequest, "Google Таблица не подключена — пришли ссылку боту: 💵 Прайс → 🌐 Google Таблица")
		return
	}
	if err != nil {
		apiErr(w, err, "источник")
		return
	}
	res := gsheet.Sync(r.Context(), src,
		func(items []domain.CatalogItem) (int, error) {
			return s.svc.BulkUpsertCatalog(r.Context(), chatID, items)
		},
		func(n int, syncErr string) error {
			return s.svc.MarkPriceSynced(r.Context(), chatID, n, syncErr)
		},
	)
	if res.Err != nil {
		writeErr(w, http.StatusBadGateway, "Синк не прошёл: "+res.Err.Error())
		return
	}
	n, _ := s.svc.CatalogCount(r.Context(), chatID)
	writeJSON(w, http.StatusOK, map[string]any{"synced": res.Count, "total": n, "duration_ms": res.Duration.Milliseconds()})
}
