//go:build ignore

// Демо-генератор share-страницы и XLSX для визуальной проверки v0.5.
// Запуск: go run scripts/gen_demo_share.go
package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"proakt/internal/domain"
	"proakt/internal/httpapi"
	"proakt/internal/store"
	"proakt/internal/xlsx"
)

type demoSvc struct{ store.Store }

func (d demoSvc) GetShareView(_ context.Context, _ string) (store.ShareView, error) {
	mk := func(pos int, name string, qty float64, unit string, price float64, hidden, done bool, note string) domain.EstimateLine {
		return domain.EstimateLine{ID: int64(pos), EstID: 3, Pos: pos, Name: name, Qty: qty, Unit: unit, Price: price, Sum: qty * price, Hidden: hidden, Done: done, Note: note}
	}
	lines := []domain.EstimateLine{
		mk(1, "укрывка окон гофрокартоном", 12, "м.п", 25, true, true, ""),
		mk(2, "заделка штроб", 47, "м.п", 60, true, true, "штробы под электрику"),
		mk(3, "грунтовка стен перед шпаклёвкой", 146.4, "м²", 25, true, true, ""),
		mk(4, "армировка откосов стекловолоконной сеткой", 18.5, "м.п", 80, true, true, ""),
		mk(5, "шлифовка стен штукатурки перед шпаклёвкой", 146.4, "м²", 50, true, true, ""),
		mk(6, "шпаклёвка стен под стеклохолст", 140.5, "м²", 140, true, true, "2 слоя"),
		mk(7, "грунтовка стен перед поклейкой стеклохолста", 75, "м²", 25, true, false, ""),
		mk(8, "поклейка стеклохолста на стены", 75, "м²", 120, false, false, ""),
		mk(9, "шпаклёвка стен под покраску по стеклохолсту", 75, "м²", 200, true, false, ""),
		mk(10, "шлифовка шпаклёвки стен под покраску", 75, "м²", 50, true, false, ""),
		mk(11, "грунтовка стен под покраску", 129.3, "м²", 25, true, false, ""),
		mk(12, "нанесение грунт-краски на стены", 129.3, "м²", 60, false, false, ""),
		mk(13, "покраска стен безвоздушным методом", 129.3, "м²", 120, false, false, "2 слоя, Dulux"),
	}
	// эмулируем store.GetShareView: суммы строк уже с коэффициентом
	for i := range lines {
		lines[i].Sum *= 1.3
	}
	var total, vis, hid float64
	for _, l := range lines {
		s := l.Sum
		total += s
		if l.Hidden {
			hid += s
		} else {
			vis += s
		}
	}
	return store.ShareView{
		ObjectName: "ЖК Сонячний, кв. 45", Title: "Малярные работы под покраску",
		Status: "approved", Note: "Смета с коэффициентом сложности 1.30 — потолки 300 см по прайс-листу.",
		Lines: lines, Photos: []store.SharePhoto{}, CreatedAt: time.Now(),
		Total: total, Visible: vis, Hidden: hid, HiddenShare: int(hid / total * 100),
	}, nil
}

func main() {
	svc := &demoSvc{}
	s := httpapi.New(httpapi.Config{BotToken: "1:demo", FilesDir: "/nonexistent"}, svc)
	srv := httptest.NewServer(s.Mux())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/s/aabbccddeeff00112233445566778899")
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 1<<20)
	n, _ := resp.Body.Read(buf)
	_ = os.WriteFile("/tmp/share_demo.html", buf[:n], 0o644)
	_ = resp.Body.Close()

	brief, _ := svc.GetShareView(context.Background(), "")
	f, _ := os.Create("/tmp/smeta_demo.xlsx")
	if err := xlsx.WriteEstimate(f, domain.EstimateBrief{
		Title: "Малярные работы под покраску", ObjectName: "ЖК Сонячний, кв. 45",
		Coeff: 1.3, Lines: len(brief.Lines), Total: brief.Total, Visible: brief.Visible,
		Hidden: brief.Hidden, HiddenShare: brief.HiddenShare,
	}, brief.Lines); err != nil {
		panic(err)
	}
	_ = f.Close()
	_ = os.Stdout.Sync()
}
