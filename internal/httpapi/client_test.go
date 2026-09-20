package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"proakt/internal/domain"
)

// authedAs — запрос с подписью конкретного Telegram user id.
func authedAs(t *testing.T, s *Server, userID int64, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	raw := signInitData(t, testToken, map[string]string{
		"user": `{"id":` + strconv.FormatInt(userID, 10) + `,"first_name":"U"}`,
	}, time.Now().Unix())
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Telegram-Init-Data", raw)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	return w
}

func TestClientOverviewForbidden(t *testing.T) {
	f := &fakeSvc{
		chat: 777,
		objs: []domain.ObjectBrief{{Object: domain.Object{ID: 1, ChatID: 777, Name: "Парковый 2", Customer: "X"}, Acts: 12, Total: 549526, Paid: 505000}},
		acts: []domain.ActBrief{{ID: 112, ActNo: 12, ObjectID: 1, Total: 64309, Paid: 19783}},
	}
	f.clientAllow = map[int64][]int64{555: {1}}
	s := New(Config{Listen: "off", BotToken: testToken}, f)

	t.Run("разрешённый видит обзор", func(t *testing.T) {
		w := authedAs(t, s, 555, "GET", "/api/client/overview?object_id=1", "")
		if w.Code != 200 {
			t.Fatalf("код %d: %s", w.Code, w.Body.String())
		}
	})
	t.Run("чужой получает 404", func(t *testing.T) {
		w := authedAs(t, s, 666, "GET", "/api/client/overview?object_id=1", "")
		if w.Code != 404 {
			t.Fatalf("хочу 404, получил %d: %s", w.Code, w.Body.String())
		}
	})
	t.Run("подмена object_id не даёт чужое", func(t *testing.T) {
		w := authedAs(t, s, 555, "GET", "/api/client/finance?object_id=2", "")
		if w.Code != 404 {
			t.Fatalf("хочу 404, получил %d", w.Code)
		}
	})
}

func TestClientGrantOwnerOnly(t *testing.T) {
	f := &fakeSvc{
		chat: 777,
		objs: []domain.ObjectBrief{{Object: domain.Object{ID: 1, ChatID: 777, Name: "Парковый 2"}}},
	}
	s := New(Config{Listen: "off", BotToken: testToken}, f)

	t.Run("владелец привязывает", func(t *testing.T) {
		w := authedAs(t, s, 777, "POST", "/api/clients/grant", `{"object_id":1,"tg_user_id":555}`)
		if w.Code != 200 {
			t.Fatalf("код %d: %s", w.Code, w.Body.String())
		}
	})
	t.Run("чужой не привязывает", func(t *testing.T) {
		w := authedAs(t, s, 666, "POST", "/api/clients/grant", `{"object_id":1,"tg_user_id":666}`)
		if w.Code != 404 {
			t.Fatalf("хочу 404, получил %d", w.Code)
		}
	})
}

func TestClientPhotoFileAccess(t *testing.T) {
	f := &fakeSvc{
		chat:         777,
		objs:         []domain.ObjectBrief{{Object: domain.Object{ID: 1, ChatID: 777, Name: "Парковый 2"}}},
		clientPhotos: []domain.PhotoRec{{ID: 1001, ObjectID: 1, ActID: nil, Caption: "Грунт"}},
	}
	f.clientAllow = map[int64][]int64{555: {1}}
	s := New(Config{Listen: "off", BotToken: testToken, FilesDir: "/nonexistent"}, f)

	t.Run("чужой не получает файл", func(t *testing.T) {
		w := authedAs(t, s, 666, "GET", "/api/client/photos/1001/file", "")
		if w.Code != 404 {
			t.Fatalf("хочу 404, получил %d", w.Code)
		}
	})
	t.Run("свой без файла на диске — честный 404", func(t *testing.T) {
		w := authedAs(t, s, 555, "GET", "/api/client/photos/1001/file", "")
		if w.Code != 404 {
			t.Fatalf("хочу 404 (нет файла), получил %d", w.Code)
		}
	})
}

func TestObjectDeleteOwnerOnly(t *testing.T) {
	f := &fakeSvc{
		chat: 777,
		objs: []domain.ObjectBrief{{Object: domain.Object{ID: 1, ChatID: 777, Name: "Парковый 2"}}},
	}
	s := New(Config{Listen: "off", BotToken: testToken}, f)

	t.Run("владелец удаляет", func(t *testing.T) {
		w := authedAs(t, s, 777, "DELETE", "/api/objects/1", "")
		if w.Code != 200 {
			t.Fatalf("код %d: %s", w.Code, w.Body.String())
		}
	})
	t.Run("чужой не удаляет", func(t *testing.T) {
		w := authedAs(t, s, 666, "DELETE", "/api/objects/1", "")
		if w.Code != 404 {
			t.Fatalf("хочу 404, получил %d", w.Code)
		}
	})
}

func TestStarterEndpoints(t *testing.T) {
	f := &fakeSvc{chat: 777}
	s := New(Config{Listen: "off", BotToken: testToken}, f)

	t.Run("статус и очистка доступны владельцу", func(t *testing.T) {
		w := authedAs(t, s, 777, "GET", "/api/starter/status", "")
		if w.Code != 200 {
			t.Fatalf("статус %d", w.Code)
		}
		w = authedAs(t, s, 777, "POST", "/api/starter/delete", "")
		if w.Code != 200 {
			t.Fatalf("очистка %d: %s", w.Code, w.Body.String())
		}
	})
}
