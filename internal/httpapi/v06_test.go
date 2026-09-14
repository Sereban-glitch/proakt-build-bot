// v06_test.go — тесты маршрутов v0.6: акт из сметы, подсказки, комментарии,
// фото строки, согласование заказчиком с уведомлением мастеру.
package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proakt/internal/domain"
)

func newV06Server(t *testing.T) (*Server, *fakeSvc, *testNotifier) {
	t.Helper()
	f := &fakeSvc{
		chat: 777,
		ests: []domain.Estimate{{ID: 5, ObjectID: 1, ChatID: 777, Title: "Кухня 9 м²", Coeff: 1}},
		estLines: map[int64][]domain.EstimateLine{
			5: {
				{ID: 51, EstID: 5, Pos: 1, Name: "штукатурка стен", Qty: 40, Unit: "м²", Price: 260, Sum: 10400},
				{ID: 52, EstID: 5, Pos: 2, Name: "грунтовка стен", Qty: 40, Unit: "м²", Price: 25, Sum: 1000},
				{ID: 53, EstID: 5, Pos: 3, Name: "покраска стен", Qty: 40, Unit: "м²", Price: 90, Sum: 3600},
				// строка-пустышка: без суммы в акт не попадёт
				{ID: 54, EstID: 5, Pos: 4, Name: "фартук (цена не заполнена)", Qty: 3, Unit: "м²", Price: 0, Sum: 0},
				// уже закрытая строка: в новый акт не попадёт
				{ID: 55, EstID: 5, Pos: 5, Name: "демонтаж", Qty: 1, Price: 2000, Sum: 2000, Done: true},
			},
		},
		comments:   map[int64][]domain.EstimateComment{},
		shareToken: "0123456789abcdef0123456789abcdef",
	}
	s := New(Config{Listen: "off", BotToken: testToken}, f)
	n := &testNotifier{}
	s.SetNotifier(n)
	return s, f, n
}

func TestEstimateActFromLines(t *testing.T) {
	s, _, _ := newV06Server(t)

	t.Run("выбор строк по id", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/estimates/5/act",
			strings.NewReader(`{"line_ids":[51,53]}`))
		req.Header.Set("X-Telegram-Init-Data", signInitData(t, testToken, map[string]string{"user": `{"id":777}`}, time.Now().Unix()))
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("ожидал 201, получил %d: %s", w.Code, w.Body.String())
		}
		var out struct {
			Act    domain.ActBrief `json:"act"`
			Closed int             `json:"closed"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		if out.Closed != 2 || out.Act.Total != 14000 {
			t.Fatalf("акт собран неверно: closed=%d total=%.2f", out.Closed, out.Act.Total)
		}
	})

	t.Run("чужая смета — 404", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/estimates/999/act", strings.NewReader(`{"all_pending":true}`))
		req.Header.Set("X-Telegram-Init-Data", signInitData(t, testToken, map[string]string{"user": `{"id":777}`}, time.Now().Unix()))
		w := httptest.NewRecorder()
		s.mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("ожидал 404 на чужую смету, получил %d", w.Code)
		}
	})
}

func TestPriceSuggest(t *testing.T) {
	s, f, _ := newV06Server(t)
	f.price = []domain.CatalogItem{
		{Name: "штукатурка стен", Unit: "м²", Price: 260},
		{Name: "штукатурка потолка", Unit: "м²", Price: 280},
		{Name: "грунтовка стен", Unit: "м²", Price: 25},
	}
	req := httptest.NewRequest("GET", "/api/price/suggest?q=штукат", nil)
	req.Header.Set("X-Telegram-Init-Data", signInitData(t, testToken, map[string]string{"user": `{"id":777}`}, time.Now().Unix()))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("код %d", w.Code)
	}
	var out struct {
		Items []domain.CatalogItem `json:"items"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Items) != 2 {
		t.Fatalf("ожидали 2 подсказки «штукатурки», получили %d: %s", len(out.Items), w.Body.String())
	}
}

func TestCommentsFlow(t *testing.T) {
	s, f, _ := newV06Server(t)

	// мастер пишет комментарий
	req := httptest.NewRequest("POST", "/api/estimates/5/comments", strings.NewReader(`{"text":"плитку можно поменять — посчитаю завтра"}`))
	req.Header.Set("X-Telegram-Init-Data", signInitData(t, testToken, map[string]string{"user": `{"id":777}`}, time.Now().Unix()))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("комментарий мастера не создался: %d %s", w.Code, w.Body.String())
	}

	// пустой комментарий — 400
	req2 := httptest.NewRequest("POST", "/api/estimates/5/comments", strings.NewReader(`{"text":"   "}`))
	req2.Header.Set("X-Telegram-Init-Data", signInitData(t, testToken, map[string]string{"user": `{"id":777}`}, time.Now().Unix()))
	w2 := httptest.NewRecorder()
	s.mux.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("ожидал 400 на пустой комментарий, получил %d", w2.Code)
	}

	// список
	req3 := httptest.NewRequest("GET", "/api/estimates/5/comments", nil)
	req3.Header.Set("X-Telegram-Init-Data", signInitData(t, testToken, map[string]string{"user": `{"id":777}`}, time.Now().Unix()))
	w3 := httptest.NewRecorder()
	s.mux.ServeHTTP(w3, req3)
	var out struct {
		Items []domain.EstimateComment `json:"items"`
	}
	_ = json.Unmarshal(w3.Body.Bytes(), &out)
	if len(out.Items) != 1 || out.Items[0].Author != "master" {
		t.Fatalf("диалог не читается: %+v", out)
	}
	_ = f
}

func TestShareApproveNotifiesMaster(t *testing.T) {
	s, f, n := newV06Server(t)

	req := httptest.NewRequest("POST", "/s/"+f.shareToken+"/approve", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("approve не прошёл: %d %s", w.Code, w.Body.String())
	}
	if len(n.approved) != 1 || n.approved[0] != "смета-1" {
		t.Fatalf("мастер не получил уведомление: %+v", n.approved)
	}

	// чужой/битый токен — 404
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/s/deadbeefdeadbeefdeadbeefdeadbeef/approve", strings.NewReader(`{}`))
	s.mux.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("чужой токен обязан быть 404, получил %d", w2.Code)
	}

	// токен с недопустимыми символами — 404 (isTokenSafe)
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("POST", "/s/ZZZZ456789abcdef0123456789abcdef/approve", strings.NewReader(`{}`))
	s.mux.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Fatalf("небезопасный токен обязан быть 404, получил %d", w3.Code)
	}
}

func TestShareCommentNotifiesMaster(t *testing.T) {
	s, f, n := newV06Server(t)
	req := httptest.NewRequest("POST", "/s/"+f.shareToken+"/comment", strings.NewReader(`{"text":"можно другую плитку?"}`))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("комментарий заказчика не создался: %d %s", w.Code, w.Body.String())
	}
	if len(n.comments) != 1 {
		t.Fatalf("мастер не узнал о вопросе: %+v", n.comments)
	}
}

func TestRoomApplyInsertsLines(t *testing.T) {
	// rooms тестируются отдельно (internal/rooms) — здесь только маршрут
	s, _, _ := newV06Server(t)
	req := httptest.NewRequest("POST", "/api/rooms/apply",
		strings.NewReader(`{"est_id":5,"preset":"kitchen","area":9,"height":2.7,"perimeter":12}`))
	req.Header.Set("X-Telegram-Init-Data", signInitData(t, testToken, map[string]string{"user": `{"id":777}`}, time.Now().Unix()))
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	// у фейка estLines[5] есть, значит маршрут должен вставить (201)
	if w.Code != http.StatusCreated {
		t.Fatalf("ожидал 201 на вставку комнаты, получил %d: %s", w.Code, w.Body.String())
	}
}
