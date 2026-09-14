package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"proakt/internal/ai"
	"proakt/internal/domain"
	"proakt/internal/store"
	"proakt/internal/webapp"
)

// Service — данные, нужные Mini App. *store.Store удовлетворяет ему неявно.
// Зависимость через интерфейс (порты-и-адаптеры): httpapi не знает о pgx,
// тесты гоняют фейковую реализацию без БД.
type Service interface {
	ListObjects(ctx context.Context, chatID int64) ([]domain.ObjectBrief, error)
	GetObject(ctx context.Context, id int64) (domain.Object, error)
	CreateObject(ctx context.Context, chatID int64, name, customer string) (domain.Object, error)
	ArchiveObject(ctx context.Context, chatID, objID int64) (string, error)

	ListActs(ctx context.Context, chatID int64, limit int) ([]domain.ActBrief, error)
	GetAct(ctx context.Context, actID int64) (domain.ActBrief, error)
	ActChatID(ctx context.Context, actID int64) (int64, error)
	ActLines(ctx context.Context, actID int64) ([]domain.ActLine, error)
	Payments(ctx context.Context, actID int64) ([]domain.Payment, error)
	DeleteAct(ctx context.Context, actID int64) (domain.ActBrief, error)
	CreatePayment(ctx context.Context, actID int64, amount float64, note string) error
	DeletePayment(ctx context.Context, chatID, payID int64) (domain.PaymentRec, error)

	ListCatalog(ctx context.Context, chatID int64) ([]domain.CatalogItem, error)
	CatalogCount(ctx context.Context, chatID int64) (int, error)
	UpsertCatalogItem(ctx context.Context, chatID int64, name, unit string, price float64) (float64, bool, error)
	DeleteCatalogItem(ctx context.Context, chatID int64, name string) (domain.CatalogItem, bool, error)

	Stats(ctx context.Context, chatID int64) (store.Stats, error)

	ListPhotos(ctx context.Context, objectID, actID int64) ([]domain.PhotoRec, error)
	GetPhoto(ctx context.Context, chatID, id int64) (domain.PhotoRec, error)
	DeletePhoto(ctx context.Context, chatID, id int64) (domain.PhotoRec, error)

	// --- сметы и шаблоны (v0.5) ---
	CreateEstimate(ctx context.Context, chatID, objectID int64, title string, coeff float64, note string) (domain.Estimate, error)
	EstimateChatID(ctx context.Context, estID int64) (int64, error)
	ListEstimates(ctx context.Context, chatID, objectID int64) ([]domain.EstimateBrief, error)
	GetEstimateBrief(ctx context.Context, chatID, estID int64) (domain.EstimateBrief, error)
	UpdateEstimate(ctx context.Context, chatID, estID int64, title, status, note *string, coeff *float64) (domain.Estimate, error)
	DeleteEstimate(ctx context.Context, chatID, estID int64) (domain.EstimateBrief, error)
	EstimateLines(ctx context.Context, estID int64) ([]domain.EstimateLine, error)
	AddEstimateLine(ctx context.Context, chatID, estID int64, name, unit string, qty, price float64, hidden bool, note string) (domain.EstimateLine, error)
	AddEstimateLinesBulk(ctx context.Context, chatID, estID int64, lines []domain.EstimateLine) (int, error)
	UpdateEstimateLine(ctx context.Context, chatID, lineID int64, name, unit, note *string, qty, price *float64, hidden, done *bool) (domain.EstimateLine, error)
	DeleteEstimateLine(ctx context.Context, chatID, lineID int64) (domain.EstimateLine, error)
	MoveEstimateLine(ctx context.Context, chatID, lineID int64, up bool) error
	SetShareToken(ctx context.Context, chatID, estID int64, token string) (string, error)
	GetShareView(ctx context.Context, token string) (store.ShareView, error)
	SharePhotoFile(ctx context.Context, token string, photoID int64) (string, error)
	ListTemplates(ctx context.Context, chatID int64) ([]domain.Template, error)
	UpsertTemplate(ctx context.Context, chatID int64, name string, lines []domain.TemplateLine) (domain.Template, error)
	DeleteTemplate(ctx context.Context, chatID int64, id int64) (domain.Template, bool, error)

	// --- v0.6: акт из сметы, фото строки, диалог, Google Таблица ---
	ActFromEstimate(ctx context.Context, chatID, estID int64, lineIDs []int64) (domain.ActBrief, int, error)
	EstimateLineOwned(ctx context.Context, chatID, lineID int64) (domain.EstimateLine, error)
	AddEstLinePhoto(ctx context.Context, chatID, estID, lineID int64, fileID, filePath, caption string) error
	ListLinePhotos(ctx context.Context, chatID, estID, lineID int64) ([]domain.PhotoRec, error)
	ListComments(ctx context.Context, chatID, estID int64) ([]domain.EstimateComment, error)
	AddComment(ctx context.Context, estID int64, author, text string) (domain.EstimateComment, error)
	PriceSource(ctx context.Context, chatID int64) (domain.PriceSource, error)
	BulkUpsertCatalog(ctx context.Context, chatID int64, items []domain.CatalogItem) (int, error)
	MarkPriceSynced(ctx context.Context, chatID int64, count int, syncErr string) error
	ApproveEstimateByToken(ctx context.Context, token string) (chatID, estID int64, title string, err error)
	ShareChatID(ctx context.Context, token string) (chatID, estID int64, title string, err error)
}

// Config — параметры HTTP-сервера.
type Config struct {
	Listen    string        // например ":8443" или "127.0.0.1:8080"
	BotToken  string        // для проверки подписи initData
	AuthTTL   time.Duration // максимум возраста initData (0 = без проверки давности)
	FilesDir  string        // где лежат фото (FILES_DIR бота)
	PublicURL string        // публичный адрес Mini App (для share-ссылок заказчику)
	Gateway   *ai.Gateway   // голос в мини-апп (nil — голос только в чате)
}

// Server — HTTP-сервер Mini App.
type Server struct {
	cfg      Config
	svc      Service
	ai       *ai.Gateway // голос (v0.6) — nil допустим: голос вернёт 503
	notifier Notifier    // уведомления мастеру (v0.6; по умолчанию заглушка)
	mux      *http.ServeMux
	srv      *http.Server
}

// New собирает сервер: /api/* — JSON, всё остальное — встроенное веб-приложение.
func New(cfg Config, svc Service) *Server {
	s := &Server{cfg: cfg, svc: svc, ai: cfg.Gateway, notifier: noopNotifier{}, mux: http.NewServeMux()}
	routes(s.mux, s)
	s.mux.Handle("GET /", webapp.Handler())
	s.srv = &http.Server{
		Addr:              cfg.Listen,
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return s
}

// Mux — доступ к маршрутизатору (тесты и встраивание в другой сервер).
func (s *Server) Mux() *http.ServeMux { return s.mux }

// Run — блокирующий запуск (как long polling в main.go). Останавливается по ctx.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("webapp: слушаю %s — Mini App доступен", s.cfg.Listen)
		errCh <- s.srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shCtx)
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// --- общие помощники ---------------------------------------------------------

type ctxKey int

const chatKey ctxKey = 1

// auth — middleware: проверяем initData, кладём chatID в контекст.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-Telegram-Init-Data")
		if raw == "" {
			if ah := r.Header.Get("Authorization"); startsWithTMA(ah) {
				raw = ah
			}
		}
		u, err := ValidateInitData(raw, s.cfg.BotToken, s.cfg.AuthTTL, time.Now())
		if err != nil {
			log.Printf("webapp: отказ авторизации с %s: %v", r.RemoteAddr, err)
			writeErr(w, http.StatusUnauthorized, "Авторизация не прошла — открой приложение из бота")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), chatKey, u.ID)))
	}
}

func startsWithTMA(s string) bool {
	return len(s) >= 4 && (s[:4] == "tma " || s[:4] == "tma\t")
}

func chatOf(r *http.Request) int64 {
	id, _ := r.Context().Value(chatKey).(int64)
	return id
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// readJSON — ограниченное чтение JSON-тела (64 КБ хватает с запасом).
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	if err := dec.Decode(dst); err != nil {
		writeErr(w, http.StatusBadRequest, "Не разобрал JSON: "+err.Error())
		return false
	}
	return true
}

func pathID(r *http.Request, name string) (int64, bool) {
	v := r.PathValue(name)
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func apiErr(w http.ResponseWriter, err error, what string) {
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, what+" не найдено")
		return
	}
	log.Printf("webapp: %s: %v", what, err)
	writeErr(w, http.StatusInternalServerError, "Ошибка базы — попробуй ещё раз")
}

// moneyF — деньги в формате «сумма до 2 знаков» (округление как в боте).
func moneyF(v float64) float64 { return float64(int64(v*100+0.5)) / 100 }
