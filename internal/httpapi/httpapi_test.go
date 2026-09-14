package httpapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proakt/internal/domain"
	"proakt/internal/store"
)

const testToken = "12345:TESTTOKEN"

// signInitData — собрать подписанный initData как это делает Telegram.
func signInitData(t *testing.T, token string, params map[string]string, authDate int64) string {
	t.Helper()
	params["auth_date"] = fmt.Sprint(authDate)
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sortStrings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	secret := hmacSHA256([]byte("WebAppData"), []byte(token))
	hash := hex.EncodeToString(hmacSHA256(secret, []byte(strings.Join(parts, "\n"))))
	parts = append(parts, "hash="+hash)
	return strings.Join(parts, "&")
}

func TestValidateInitData(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	user := `{"id":777,"first_name":"Серёга","username":"master"}`
	good := map[string]string{"user": user, "query_id": "AAF"}

	t.Run("корректная подпись", func(t *testing.T) {
		raw := signInitData(t, testToken, cloneMap(good), now.Unix())
		u, err := ValidateInitData(raw, testToken, 24*time.Hour, now)
		if err != nil {
			t.Fatalf("не ожидал ошибки: %v", err)
		}
		if u.ID != 777 || u.FirstName != "Серёга" {
			t.Fatalf("не тот пользователь: %+v", u)
		}
	})

	t.Run("подделанный hash", func(t *testing.T) {
		raw := signInitData(t, testToken, cloneMap(good), now.Unix())
		raw = strings.Replace(raw, "user=", "user%22x%22=&user=", 1) // ломаем содержимое
		raw = strings.Replace(raw, "hash=", "hash=0", 1)
		if _, err := ValidateInitData(raw, testToken, 24*time.Hour, now); err == nil {
			t.Fatal("ожидал отказ по подписи")
		}
	})

	t.Run("чужой токен бота", func(t *testing.T) {
		raw := signInitData(t, "1:WRONG", cloneMap(good), now.Unix())
		if _, err := ValidateInitData(raw, testToken, 24*time.Hour, now); err == nil {
			t.Fatal("подпись другим токеном обязана не пройти")
		}
	})

	t.Run("устаревший auth_date", func(t *testing.T) {
		raw := signInitData(t, testToken, cloneMap(good), now.Add(-30*time.Hour).Unix())
		if _, err := ValidateInitData(raw, testToken, 24*time.Hour, now); err == nil {
			t.Fatal("старый initData обязан отклоняться (replay)")
		}
	})

	t.Run("prefix tma в Authorization", func(t *testing.T) {
		raw := signInitData(t, testToken, cloneMap(good), now.Unix())
		u, err := ValidateInitData("tma "+raw, testToken, 24*time.Hour, now)
		if err != nil || u.ID != 777 {
			t.Fatalf("схема tma не разобралась: %v", err)
		}
	})
}

func TestValidateInitData2026(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	user := `{"id":777,"first_name":"Серёга","username":"master"}`
	good := map[string]string{"user": user, "query_id": "AAF", "signature": "qwerty123"}
	raw := signInitData(t, testToken, cloneMap(good), now.Unix())
	u, err := ValidateInitData(raw, testToken, 24*time.Hour, now)
	if err != nil {
		t.Fatalf("2026-формат: hash считается ВМЕСТЕ с signature — обязано пройти: %v", err)
	}
	if u.ID != 777 {
		t.Fatalf("не тот пользователь: %+v", u)
	}
}

func cloneMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// fakeSvc — фейковый Service: без БД, данные одного «чата» 777.
type fakeSvc struct {
	chat  int64
	objs  []domain.ObjectBrief
	acts  []domain.ActBrief
	lines []domain.ActLine
	pays  []domain.Payment
	price []domain.CatalogItem

	estLineSeq int64
	ests       []domain.Estimate
	estLines   map[int64][]domain.EstimateLine // estID -> lines
	tpls       []domain.Template
	tplSeq     int64

	actSeq     int64
	createdAct domain.ActBrief
	linePhotos []string
	comments   map[int64][]domain.EstimateComment
	shareToken string
}

func (f *fakeSvc) ListObjects(_ context.Context, chatID int64) ([]domain.ObjectBrief, error) {
	if chatID != f.chat {
		return nil, nil
	}
	return f.objs, nil
}
func (f *fakeSvc) GetObject(_ context.Context, id int64) (domain.Object, error) {
	for _, o := range f.objs {
		if o.ID == id {
			return o.Object, nil
		}
	}
	return domain.Object{}, store.ErrNotFound
}
func (f *fakeSvc) CreateObject(_ context.Context, chatID int64, name, customer string) (domain.Object, error) {
	return domain.Object{ID: 42, ChatID: chatID, Name: name, Customer: customer}, nil
}
func (f *fakeSvc) ArchiveObject(_ context.Context, chatID, objID int64) (string, error) {
	if chatID != f.chat {
		return "", store.ErrNotFound
	}
	return "объект-1", nil
}
func (f *fakeSvc) ListActs(_ context.Context, chatID int64, _ int) ([]domain.ActBrief, error) {
	if chatID != f.chat {
		return nil, nil
	}
	return f.acts, nil
}
func (f *fakeSvc) GetAct(_ context.Context, actID int64) (domain.ActBrief, error) {
	for _, a := range f.acts {
		if a.ID == actID {
			return a, nil
		}
	}
	return domain.ActBrief{}, store.ErrNotFound
}
func (f *fakeSvc) ActChatID(_ context.Context, actID int64) (int64, error) {
	for _, a := range f.acts {
		if a.ID == actID {
			return f.chat, nil
		}
	}
	return 0, store.ErrNotFound
}
func (f *fakeSvc) ActLines(_ context.Context, _ int64) ([]domain.ActLine, error) { return f.lines, nil }
func (f *fakeSvc) Payments(_ context.Context, _ int64) ([]domain.Payment, error) { return f.pays, nil }
func (f *fakeSvc) DeleteAct(_ context.Context, actID int64) (domain.ActBrief, error) {
	for i, a := range f.acts {
		if a.ID == actID {
			f.acts = append(f.acts[:i], f.acts[i+1:]...)
			return a, nil
		}
	}
	return domain.ActBrief{}, store.ErrNotFound
}
func (f *fakeSvc) CreatePayment(_ context.Context, _ int64, _ float64, _ string) error { return nil }
func (f *fakeSvc) DeletePayment(_ context.Context, chatID, _ int64) (domain.PaymentRec, error) {
	if chatID != f.chat {
		return domain.PaymentRec{}, store.ErrNotFound
	}
	return domain.PaymentRec{Amount: 100}, nil
}
func (f *fakeSvc) ListCatalog(_ context.Context, chatID int64) ([]domain.CatalogItem, error) {
	if chatID != f.chat {
		return nil, nil
	}
	return f.price, nil
}
func (f *fakeSvc) CatalogCount(_ context.Context, chatID int64) (int, error) {
	return len(f.price), nil
}
func (f *fakeSvc) UpsertCatalogItem(_ context.Context, _ int64, name, unit string, price float64) (float64, bool, error) {
	return 240, true, nil
}
func (f *fakeSvc) DeleteCatalogItem(_ context.Context, _ int64, _ string) (domain.CatalogItem, bool, error) {
	return domain.CatalogItem{}, false, nil
}
func (f *fakeSvc) Stats(_ context.Context, chatID int64) (store.Stats, error) {
	if chatID != f.chat {
		return store.Stats{}, nil
	}
	return store.Stats{Objects: 1, Acts: 2, Total: 11700, Paid: 10000, UnpaidActs: 1}, nil
}
func (f *fakeSvc) ListPhotos(_ context.Context, _, _ int64) ([]domain.PhotoRec, error) {
	return nil, nil
}
func (f *fakeSvc) GetPhoto(_ context.Context, chatID, id int64) (domain.PhotoRec, error) {
	return domain.PhotoRec{}, store.ErrNotFound
}
func (f *fakeSvc) DeletePhoto(_ context.Context, chatID, id int64) (domain.PhotoRec, error) {
	return domain.PhotoRec{}, store.ErrNotFound
}

// --- v0.6: фейковые методы для новых маршрутов ---------------------------------

func (f *fakeSvc) ActFromEstimate(_ context.Context, chatID, estID int64, lineIDs []int64) (domain.ActBrief, int, error) {
	if chatID != f.chat {
		return domain.ActBrief{}, 0, store.ErrNotFound
	}
	_, ok := f.estLines[estID]
	if !ok {
		return domain.ActBrief{}, 0, store.ErrNotFound
	}
	var picked []domain.EstimateLine
	for _, l := range f.estLines[estID] {
		if !l.Done && (len(lineIDs) == 0 || containsID(lineIDs, l.ID)) {
			if l.Sum > 0.009 || l.Price > 0.009 {
				picked = append(picked, l)
			}
		}
	}
	if len(picked) == 0 {
		return domain.ActBrief{}, 0, store.ErrNotFound
	}
	var total float64
	for _, l := range picked {
		total += l.Sum
	}
	f.actSeq++
	brief := domain.ActBrief{ID: 900 + f.actSeq, ActNo: int(f.actSeq), ObjectID: 1, ObjectName: "объект", Total: total}
	f.createdAct = brief
	return brief, len(picked), nil
}

func (f *fakeSvc) EstimateLineOwned(_ context.Context, chatID, lineID int64) (domain.EstimateLine, error) {
	if chatID != f.chat {
		return domain.EstimateLine{}, store.ErrNotFound
	}
	for _, ls := range f.estLines {
		for _, l := range ls {
			if l.ID == lineID {
				return l, nil
			}
		}
	}
	return domain.EstimateLine{}, store.ErrNotFound
}

func (f *fakeSvc) AddEstLinePhoto(_ context.Context, chatID, estID, lineID int64, _, path, caption string) error {
	if chatID != f.chat {
		return store.ErrNotFound
	}
	f.linePhotos = append(f.linePhotos, path+"|"+caption)
	return nil
}

func (f *fakeSvc) ListLinePhotos(_ context.Context, chatID, _, _ int64) ([]domain.PhotoRec, error) {
	if chatID != f.chat {
		return nil, nil
	}
	return nil, nil
}

func (f *fakeSvc) ListComments(_ context.Context, chatID, estID int64) ([]domain.EstimateComment, error) {
	if chatID != f.chat {
		return nil, nil
	}
	return f.comments[estID], nil
}

func (f *fakeSvc) AddComment(_ context.Context, estID int64, author, text string) (domain.EstimateComment, error) {
	c := domain.EstimateComment{ID: int64(len(f.comments[estID]) + 1), EstID: estID, Author: author, Text: text}
	f.comments[estID] = append(f.comments[estID], c)
	return c, nil
}

func (f *fakeSvc) PriceSource(_ context.Context, chatID int64) (domain.PriceSource, error) {
	if chatID != f.chat {
		return domain.PriceSource{}, store.ErrNotFound
	}
	return domain.PriceSource{ChatID: chatID, URL: "https://docs.google.com/spreadsheets/d/X/edit", FileID: "X"}, nil
}

func (f *fakeSvc) BulkUpsertCatalog(_ context.Context, chatID int64, items []domain.CatalogItem) (int, error) {
	if chatID != f.chat {
		return 0, store.ErrNotFound
	}
	return len(items), nil
}

func (f *fakeSvc) MarkPriceSynced(_ context.Context, _ int64, _ int, _ string) error { return nil }

func (f *fakeSvc) ApproveEstimateByToken(_ context.Context, token string) (int64, int64, string, error) {
	if token != f.shareToken {
		return 0, 0, "", store.ErrNotFound
	}
	return f.chat, 1, "смета-1", nil
}

func (f *fakeSvc) ShareChatID(_ context.Context, token string) (int64, int64, string, error) {
	if token != f.shareToken {
		return 0, 0, "", store.ErrNotFound
	}
	return f.chat, 1, "смета-1", nil
}

func containsID(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// testNotifier — ловит уведомления мастеру (v0.6).
type testNotifier struct {
	approved []string
	comments []string
}

func (n *testNotifier) NotifyEstimateApproved(_ int64, title string) {
	n.approved = append(n.approved, title)
}
func (n *testNotifier) NotifyClientComment(_ int64, title, _ string) {
	n.comments = append(n.comments, title)
}

// authedRequest — запрос с валидным initData чата 777.
func authedRequest(t *testing.T, s *Server, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	now := time.Now()
	raw := signInitData(t, testToken, map[string]string{
		"user": `{"id":777,"first_name":"Серёга"}`,
	}, now.Unix())
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("X-Telegram-Init-Data", raw)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	return w
}

func TestServerAuthAndRouting(t *testing.T) {
	f := &fakeSvc{
		chat:  777,
		objs:  []domain.ObjectBrief{{Object: domain.Object{ID: 1, ChatID: 777, Name: "ЖК Сонячний", Customer: "Иван"}, Acts: 2, Total: 11700, Paid: 10000, Photos: 3}},
		acts:  []domain.ActBrief{{ID: 10, ActNo: 1, ObjectID: 1, ObjectName: "ЖК Сонячний", Total: 11700, Paid: 10000}},
		lines: []domain.ActLine{{Pos: 1, Name: "штукатурка", Qty: 45, Unit: "м²", Price: 260, Sum: 11700}},
		pays:  []domain.Payment{{ID: 5, ActID: 10, Amount: 10000}},
		price: []domain.CatalogItem{{Name: "штукатурка", Unit: "м²", Price: 260}},
	}
	s := New(Config{Listen: "off", BotToken: testToken}, f)

	t.Run("без авторизации — 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dashboard", nil)
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("ожидал 401, получил %d", w.Code)
		}
	})

	t.Run("дашборд отдаёт сводку", func(t *testing.T) {
		w := authedRequest(t, s, "GET", "/api/dashboard")
		if w.Code != 200 {
			t.Fatalf("код %d: %s", w.Code, w.Body.String())
		}
		var out struct {
			Stats     store.Stats       `json:"stats"`
			DebtTotal float64           `json:"debt_total"`
			Debts     []domain.ActBrief `json:"debts"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.Stats.Total != 11700 || out.DebtTotal != 1700 || len(out.Debts) != 1 {
			t.Fatalf("неверные данные: %+v", out)
		}
	})

	t.Run("чужой chat_id видит пустоту", func(t *testing.T) {
		now := time.Now()
		raw := signInitData(t, testToken, map[string]string{
			"user": `{"id":666,"first_name":"Чужак"}`,
		}, now.Unix())
		req := httptest.NewRequest("GET", "/api/objects", nil)
		req.Header.Set("X-Telegram-Init-Data", raw)
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "[]") {
			t.Fatalf("чужой получил не пустой список: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("создание объекта валидирует имя", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/objects", strings.NewReader(`{"name":"Ж"}`))
		sign := signInitData(t, testToken, map[string]string{"user": `{"id":777}`}, time.Now().Unix())
		req.Header.Set("X-Telegram-Init-Data", sign)
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("ожидал 400 на короткое имя, получил %d", w.Code)
		}
	})
}
