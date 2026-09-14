// estimates_test.go — тесты REST-слоя смет и шаблонов (v0.5).
// Тот же подход, что httpapi_test.go: фейковый Service, без БД и сети.
// Проверяем: чат-скоуп (IDOR), валидацию, полный CRUD, share-токены, парсер.
package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"proakt/internal/domain"
	"proakt/internal/store"
)

// --- фейковые методы смет -----------------------------------------------------

func (f *fakeSvc) CreateEstimate(_ context.Context, chatID, objectID int64, title string, coeff float64, note string) (domain.Estimate, error) {
	found := false
	for _, o := range f.objs {
		if o.ID == objectID {
			found = true
		}
	}
	if !found || chatID != f.chat {
		return domain.Estimate{}, store.ErrNotFound
	}
	e := domain.Estimate{ID: int64(len(f.ests) + 1), ObjectID: objectID, ChatID: chatID, Title: title, Coeff: coeff, Note: note}
	f.ests = append(f.ests, e)
	return e, nil
}

func (f *fakeSvc) EstimateChatID(_ context.Context, estID int64) (int64, error) {
	for _, e := range f.ests {
		if e.ID == estID {
			return e.ChatID, nil
		}
	}
	return 0, store.ErrNotFound
}

func (f *fakeSvc) ListEstimates(_ context.Context, chatID, objectID int64) ([]domain.EstimateBrief, error) {
	if chatID != f.chat {
		return nil, nil
	}
	var out []domain.EstimateBrief
	for _, e := range f.ests {
		if objectID > 0 && e.ObjectID != objectID {
			continue
		}
		out = append(out, domain.EstimateBrief{ID: e.ID, ObjectID: e.ObjectID, Title: e.Title, Coeff: e.Coeff})
	}
	return out, nil
}

func (f *fakeSvc) GetEstimateBrief(_ context.Context, chatID, estID int64) (domain.EstimateBrief, error) {
	for _, e := range f.ests {
		if e.ID == estID && e.ChatID == chatID {
			b := domain.EstimateBrief{ID: e.ID, ObjectID: e.ObjectID, Title: e.Title, Coeff: e.Coeff, Status: e.Status, ShareToken: e.ShareToken}
			for _, l := range f.estLines[estID] {
				b.Lines++
				b.Total += l.Sum * e.Coeff
				if l.Hidden {
					b.Hidden += l.Sum * e.Coeff
				} else {
					b.Visible += l.Sum * e.Coeff
				}
				if l.Done {
					b.DoneSum += l.Sum * e.Coeff
				}
			}
			if b.Total > 0.01 {
				b.HiddenShare = int(b.Hidden/b.Total*100 + 0.5)
			}
			return b, nil
		}
	}
	return domain.EstimateBrief{}, store.ErrNotFound
}

func (f *fakeSvc) UpdateEstimate(_ context.Context, chatID, estID int64, title, status, note *string, coeff *float64) (domain.Estimate, error) {
	for i, e := range f.ests {
		if e.ID == estID && e.ChatID == chatID {
			if title != nil && *title != "" {
				e.Title = *title
			}
			if status != nil {
				e.Status = *status
			}
			if note != nil {
				e.Note = *note
			}
			if coeff != nil {
				e.Coeff = *coeff
			}
			f.ests[i] = e
			return e, nil
		}
	}
	return domain.Estimate{}, store.ErrNotFound
}

func (f *fakeSvc) DeleteEstimate(_ context.Context, chatID, estID int64) (domain.EstimateBrief, error) {
	b, err := f.GetEstimateBrief(context.Background(), chatID, estID)
	if err != nil {
		return b, err
	}
	for i, e := range f.ests {
		if e.ID == estID && e.ChatID == chatID {
			f.ests = append(f.ests[:i], f.ests[i+1:]...)
			delete(f.estLines, estID)
			return b, nil
		}
	}
	return b, store.ErrNotFound
}

func (f *fakeSvc) EstimateLines(_ context.Context, estID int64) ([]domain.EstimateLine, error) {
	return f.estLines[estID], nil
}

func (f *fakeSvc) AddEstimateLine(_ context.Context, chatID, estID int64, name, unit string, qty, price float64, hidden bool, note string) (domain.EstimateLine, error) {
	if _, err := f.getEst(chatID, estID); err != nil {
		return domain.EstimateLine{}, store.ErrNotFound
	}
	f.estLineSeq++
	l := domain.EstimateLine{ID: f.estLineSeq, EstID: estID, Pos: len(f.estLines[estID]) + 1,
		Name: name, Unit: unit, Qty: qty, Price: price, Sum: qty * price, Hidden: hidden, Note: note}
	f.estLines[estID] = append(f.estLines[estID], l)
	return l, nil
}

func (f *fakeSvc) AddEstimateLinesBulk(_ context.Context, chatID, estID int64, lines []domain.EstimateLine) (int, error) {
	if _, err := f.getEst(chatID, estID); err != nil {
		return 0, store.ErrNotFound
	}
	for _, l := range lines {
		if _, err := f.AddEstimateLine(context.Background(), chatID, estID, l.Name, l.Unit, l.Qty, l.Price, l.Hidden, l.Note); err != nil {
			return 0, err
		}
	}
	return len(lines), nil
}

func (f *fakeSvc) UpdateEstimateLine(_ context.Context, chatID, lineID int64, name, unit, note *string, qty, price *float64, hidden, done *bool) (domain.EstimateLine, error) {
	l, err := f.lineOwned(chatID, lineID)
	if err != nil {
		return l, err
	}
	if name != nil && *name != "" {
		l.Name = *name
	}
	if unit != nil {
		l.Unit = *unit
	}
	if note != nil {
		l.Note = *note
	}
	if qty != nil {
		l.Qty = *qty
	}
	if price != nil {
		l.Price = *price
	}
	if hidden != nil {
		l.Hidden = *hidden
	}
	if done != nil {
		l.Done = *done
	}
	l.Sum = l.Qty * l.Price
	list := f.estLines[l.EstID]
	for i := range list {
		if list[i].ID == lineID {
			list[i] = l
		}
	}
	return l, nil
}

func (f *fakeSvc) DeleteEstimateLine(_ context.Context, chatID, lineID int64) (domain.EstimateLine, error) {
	l, err := f.lineOwned(chatID, lineID)
	if err != nil {
		return l, err
	}
	list := f.estLines[l.EstID]
	for i := range list {
		if list[i].ID == lineID {
			f.estLines[l.EstID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return l, nil
}

func (f *fakeSvc) MoveEstimateLine(_ context.Context, chatID, lineID int64, up bool) error {
	l, err := f.lineOwned(chatID, lineID)
	if err != nil {
		return err
	}
	list := f.estLines[l.EstID]
	i := -1
	for j := range list {
		if list[j].ID == lineID {
			i = j
		}
	}
	j := i + 1
	if up {
		j = i - 1
	}
	if i < 0 || j < 0 || j >= len(list) {
		return store.ErrNotFound
	}
	list[i], list[j] = list[j], list[i]
	return nil
}

func (f *fakeSvc) SetShareToken(_ context.Context, chatID, estID int64, token string) (string, error) {
	if _, err := f.getEst(chatID, estID); err != nil {
		return "", err
	}
	for i, e := range f.ests {
		if e.ID == estID && e.ChatID == chatID {
			e.ShareToken = token
			f.ests[i] = e
			return token, nil
		}
	}
	return "", store.ErrNotFound
}

func (f *fakeSvc) GetShareView(_ context.Context, token string) (store.ShareView, error) {
	for _, e := range f.ests {
		if e.ShareToken == token && token != "" {
			v := store.ShareView{Title: e.Title, Status: e.Status, Lines: []domain.EstimateLine{}}
			for _, o := range f.objs {
				if o.ID == e.ObjectID {
					v.ObjectName = o.Name
				}
			}
			for _, l := range f.estLines[e.ID] {
				v.Lines = append(v.Lines, l)
				v.Total += l.Sum * e.Coeff
				if l.Hidden {
					v.Hidden += l.Sum * e.Coeff
				} else {
					v.Visible += l.Sum * e.Coeff
				}
			}
			if v.Total > 0.01 {
				v.HiddenShare = int(v.Hidden/v.Total*100 + 0.5)
			}
			return v, nil
		}
	}
	return store.ShareView{}, store.ErrNotFound
}

func (f *fakeSvc) SharePhotoFile(_ context.Context, _ string, _ int64) (string, error) {
	return "", store.ErrNotFound
}

func (f *fakeSvc) ListTemplates(_ context.Context, chatID int64) ([]domain.Template, error) {
	if chatID != f.chat {
		return nil, nil
	}
	return f.tpls, nil
}

func (f *fakeSvc) UpsertTemplate(_ context.Context, chatID int64, name string, lines []domain.TemplateLine) (domain.Template, error) {
	if chatID != f.chat {
		return domain.Template{}, store.ErrNotFound
	}
	for i, t := range f.tpls {
		if t.Name == name {
			f.tpls[i].Lines = lines
			return f.tpls[i], nil
		}
	}
	f.tplSeq++
	t := domain.Template{ID: f.tplSeq, Name: name, Lines: lines}
	f.tpls = append(f.tpls, t)
	return t, nil
}

func (f *fakeSvc) DeleteTemplate(_ context.Context, chatID int64, id int64) (domain.Template, bool, error) {
	if chatID != f.chat {
		return domain.Template{}, false, nil
	}
	for i, t := range f.tpls {
		if t.ID == id {
			f.tpls = append(f.tpls[:i], f.tpls[i+1:]...)
			return t, true, nil
		}
	}
	return domain.Template{}, false, nil
}

func (f *fakeSvc) getEst(chatID, estID int64) (domain.Estimate, error) {
	for _, e := range f.ests {
		if e.ID == estID && e.ChatID == chatID {
			return e, nil
		}
	}
	return domain.Estimate{}, store.ErrNotFound
}

func (f *fakeSvc) lineOwned(chatID, lineID int64) (domain.EstimateLine, error) {
	for _, lines := range f.estLines {
		for _, l := range lines {
			if l.ID == lineID {
				if _, err := f.getEst(chatID, l.EstID); err == nil {
					return l, nil
				}
			}
		}
	}
	return domain.EstimateLine{}, store.ErrNotFound
}

// --- утилиты запросов ----------------------------------------------------------

// seedEstimate — сервер + смета с двумя позициями у чата 777.
func seedEstimate(t *testing.T) (*httptest.Server, *fakeSvc, int64) {
	t.Helper()
	f := &fakeSvc{chat: 777, objs: []domain.ObjectBrief{{Object: domain.Object{ID: 5, ChatID: 777, Name: "Парковий 2"}}}}
	f.estLines = map[int64][]domain.EstimateLine{}
	s := New(Config{BotToken: testToken}, f)
	srv := httptest.NewServer(s.mux) // mux тот же, что в production-роутинге
	t.Cleanup(srv.Close)
	e, err := f.CreateEstimate(context.Background(), 777, 5, "Малярные работы", 1.3, "потолки 300 см")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.AddEstimateLine(context.Background(), 777, e.ID, "грунтовка стен", "м²", 129.3, 25, true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.AddEstimateLine(context.Background(), 777, e.ID, "покраска стен", "м²", 129.3, 120, false, ""); err != nil {
		t.Fatal(err)
	}
	return srv, f, e.ID
}

// apiReq — HTTP-запрос к живому тест-серверу с подписью initData чата chatID.
func apiReq(t *testing.T, base, method, target, body string, chatID int64) (*http.Response, string) {
	t.Helper()
	raw := signInitData(t, testToken, map[string]string{
		"user": fmt.Sprintf(`{"id":%d,"first_name":"Мастер"}`, chatID),
	}, time.Now().Unix())
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, base+target, rd)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Telegram-Init-Data", raw)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp, string(b)
}

func id64(v int64) string { return strconv.FormatInt(v, 10) }

// --- сами тесты ---------------------------------------------------------------

func TestEstimateCRUDFlow(t *testing.T) {
	srv, f, estID := seedEstimate(t)
	base, eid := srv.URL, id64(estID)

	// список
	resp, body := apiReq(t, base, "GET", "/api/estimates", "", 777)
	if resp.StatusCode != 200 || !strings.Contains(body, "Малярные работы") {
		t.Fatalf("список: %d %s", resp.StatusCode, body)
	}

	// детальная с деньгами: грунт 129.3*25=3232.5 → ×1.3 = 4202.25
	resp, body = apiReq(t, base, "GET", "/api/estimates/"+eid, "", 777)
	if resp.StatusCode != 200 {
		t.Fatalf("детали: %d", resp.StatusCode)
	}
	if !strings.Contains(body, "4202.25") || !strings.Contains(body, `"hidden"`) {
		t.Fatalf("нет пересчёта с коэффициентом: %s", body)
	}

	// добавить позицию
	resp, body = apiReq(t, base, "POST", "/api/estimates/"+eid+"/lines",
		`{"name":"шлифовка стен","qty":146.4,"unit":"м²","price":50,"hidden":true}`, 777)
	if resp.StatusCode != 201 {
		t.Fatalf("добавление: %d %s", resp.StatusCode, body)
	}

	// bulk (шаблон/мультистрока)
	resp, body = apiReq(t, base, "POST", "/api/estimates/"+eid+"/lines",
		`{"lines":[{"name":"поклейка стеклохолста","qty":75,"unit":"м²","price":120},{"name":"шпаклёвка под покраску","qty":75,"unit":"м²","price":200,"hidden":true}]}`, 777)
	if resp.StatusCode != 201 || !strings.Contains(body, `"added":2`) {
		t.Fatalf("bulk: %d %s", resp.StatusCode, body)
	}

	// патч строки: done=true
	lineID := f.estLines[estID][1].ID
	resp, body = apiReq(t, base, "PATCH", "/api/estimates/"+eid+"/lines/"+id64(lineID), `{"done":true}`, 777)
	if resp.StatusCode != 200 || !strings.Contains(body, `"done":true`) {
		t.Fatalf("патч строки: %d %s", resp.StatusCode, body)
	}

	// move
	resp, _ = apiReq(t, base, "POST", "/api/estimates/"+eid+"/lines/"+id64(lineID)+"/move", `{"dir":"up"}`, 777)
	if resp.StatusCode != 200 {
		t.Fatalf("move: %d", resp.StatusCode)
	}

	// share: токен выдаётся, повторный вызов перевыпускает
	resp, body = apiReq(t, base, "POST", "/api/estimates/"+eid+"/share", "", 777)
	if resp.StatusCode != 200 || !strings.Contains(body, `"token"`) {
		t.Fatalf("share: %d %s", resp.StatusCode, body)
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(body), &m)
	tok1, _ := m["token"].(string)
	if len(tok1) != 32 {
		t.Fatalf("токен подозрительный: %q", tok1)
	}
	resp, body = apiReq(t, base, "POST", "/api/estimates/"+eid+"/share", "", 777)
	_ = json.Unmarshal([]byte(body), &m)
	if m["token"] == tok1 {
		t.Fatal("токен не перевыпустился")
	}

	// валидация: плохой коэффициент
	resp, _ = apiReq(t, base, "PATCH", "/api/estimates/"+eid, `{"coeff":9}`, 777)
	if resp.StatusCode != 400 {
		t.Fatalf("коэфф 9 принят: %d", resp.StatusCode)
	}

	// удаление позиции и сметы
	resp, _ = apiReq(t, base, "DELETE", "/api/estimates/"+eid+"/lines/"+id64(lineID), "", 777)
	if resp.StatusCode != 200 {
		t.Fatalf("delete line: %d", resp.StatusCode)
	}
	resp, _ = apiReq(t, base, "DELETE", "/api/estimates/"+eid, "", 777)
	if resp.StatusCode != 200 {
		t.Fatalf("delete сметы: %d", resp.StatusCode)
	}
	if len(f.ests) != 0 || len(f.estLines[estID]) != 0 {
		t.Fatal("смета/строки не удалены из фейка")
	}
}

func TestEstimateOwnership(t *testing.T) {
	srv, _, estID := seedEstimate(t)
	base, eid := srv.URL, id64(estID)

	// чужой чат (999) — всё 404
	resp, _ := apiReq(t, base, "GET", "/api/estimates/"+eid, "", 999)
	if resp.StatusCode != 404 {
		t.Fatalf("чужая смета читается: %d", resp.StatusCode)
	}
	resp, _ = apiReq(t, base, "DELETE", "/api/estimates/"+eid, "", 999)
	if resp.StatusCode != 404 {
		t.Fatalf("чужая смета удаляется: %d", resp.StatusCode)
	}
	resp, _ = apiReq(t, base, "POST", "/api/estimates/"+eid+"/lines", `{"name":"взлом","qty":1,"price":1}`, 999)
	if resp.StatusCode != 404 {
		t.Fatalf("чужая смета принимает строки: %d", resp.StatusCode)
	}
}

func TestEstimateValidation(t *testing.T) {
	srv, _, estID := seedEstimate(t)
	base := srv.URL

	cases := []struct{ body, url string }{
		{`{"name":"x","qty":1,"price":1}`, "/api/estimates/" + id64(estID) + "/lines"},   // имя 1 символ
		{`{"name":"ок","qty":-5,"price":1}`, "/api/estimates/" + id64(estID) + "/lines"}, // отриц. количество
		{`{"name":"ок","qty":1,"price":-3}`, "/api/estimates/" + id64(estID) + "/lines"}, // отриц. цена
		{`{"dir":"sideways"}`, "/api/estimates/" + id64(estID) + "/lines/1/move"},        // кривой dir
	}
	for _, c := range cases {
		resp, body := apiReq(t, base, "POST", c.url, c.body, 777)
		if resp.StatusCode != 400 {
			t.Fatalf("тело %s на %s: код %d, ждали 400 — %s", c.body, c.url, resp.StatusCode, body)
		}
	}
}

func TestParseLineEndpoint(t *testing.T) {
	srv, _, _ := seedEstimate(t)

	resp, body := apiReq(t, srv.URL, "POST", "/api/parse-line", `{"text":"штукатурка 45 м² 260"}`, 777)
	if resp.StatusCode != 200 || !strings.Contains(body, "штукатурка") {
		t.Fatalf("parse: %d %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `"qty":45`) {
		t.Fatalf("количество не распознано: %s", body)
	}

	// подозрительная мультистрока — предупреждение, не 500
	resp, body = apiReq(t, srv.URL, "POST", "/api/parse-line", `{"text":"шлифовка 45 грунтовка 45"}`, 777)
	if resp.StatusCode != 200 || !strings.Contains(body, "suspicious") {
		t.Fatalf("suspicious: %d %s", resp.StatusCode, body)
	}
}

func TestTemplatesAPI(t *testing.T) {
	srv, f, _ := seedEstimate(t)

	body := `{"name":"Покраска комнаты","lines":[` +
		`{"name":"грунтовка стен","qty":1,"unit":"м²","price":25,"hidden":true},` +
		`{"name":"покраска стен","qty":1,"unit":"м²","price":120}]}`
	resp, body := apiReq(t, srv.URL, "POST", "/api/templates", body, 777)
	if resp.StatusCode != 200 || !strings.Contains(body, `"id":1`) {
		t.Fatalf("upsert: %d %s", resp.StatusCode, body)
	}

	// обновление по имени — не плодит дубли
	resp, _ = apiReq(t, srv.URL, "POST", "/api/templates", body, 777)
	if resp.StatusCode != 200 || len(f.tpls) != 1 {
		t.Fatalf("не обновился по имени: %d, шаблонов %d", resp.StatusCode, len(f.tpls))
	}

	resp, body = apiReq(t, srv.URL, "GET", "/api/templates", "", 777)
	if resp.StatusCode != 200 || !strings.Contains(body, "Покраска комнаты") {
		t.Fatalf("список: %d %s", resp.StatusCode, body)
	}

	resp, _ = apiReq(t, srv.URL, "DELETE", "/api/templates/1", "", 777)
	if resp.StatusCode != 200 {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	resp, _ = apiReq(t, srv.URL, "DELETE", "/api/templates/1", "", 777)
	if resp.StatusCode != 404 {
		t.Fatalf("повторное delete: %d", resp.StatusCode)
	}
}

func TestSharePageRenders(t *testing.T) {
	srv, f, estID := seedEstimate(t)

	_, err := f.SetShareToken(context.Background(), 777, estID, "aabbccddeeff00112233445566778899")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(srv.URL + "/s/aabbccddeeff00112233445566778899")
	if err != nil {
		t.Fatal(err)
	}
	html, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("share page: %d", resp.StatusCode)
	}
	for _, want := range []string{"Малярные работы", "Парковий 2", "грн", "Скрытая подготовка"} {
		if !strings.Contains(string(html), want) {
			t.Fatalf("в странице нет %q", want)
		}
	}
	if resp.Header.Get("X-Robots-Tag") == "" {
		t.Fatal("нет X-Robots-Tag")
	}

	resp2, err := http.Get(srv.URL + "/s/zzzzzzzzzzzzzzzz")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if resp2.StatusCode != 404 {
		t.Fatalf("кривой токен: %d", resp2.StatusCode)
	}
}
