// Тесты v0.3.7: «упущенная выгода» в отчёте, подписи фото, кнопки показа.
package app

import (
	"strings"
	"testing"
	"time"

	"proakt/internal/domain"
	"proakt/internal/store"
)

func TestReportTextMissedProfit(t *testing.T) {
	// живой кейс мастера: акт №1 11 700 не оплачен, акт №2 = 0 грн
	st := store.Stats{Objects: 1, Acts: 2, Total: 11700, Paid: 0, Photos: 0,
		ZeroActs: 1, UnpaidActs: 1}
	txt := reportText(st, 0)
	for _, want := range []string{
		"Упущенная выгода",
		"не оплачено 11 700 грн (1 акт)",
		"1 акт с суммой 0",
		"деньги не выставлены",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("отчёт без %q:\n%s", want, txt)
		}
	}
	if strings.Contains(txt, "Долг: ") {
		t.Errorf("старую строку «Долг:» заменили блоком упущенной выгоды:\n%s", txt)
	}
}

func TestReportTextAllClean(t *testing.T) {
	st := store.Stats{Objects: 2, Acts: 3, Total: 40000, Paid: 40000, Photos: 5,
		ZeroActs: 0, UnpaidActs: 0}
	txt := reportText(st, 70)
	for _, want := range []string{"Долгов нет", "деньги под контролем", "📷 Фото: 5"} {
		if !strings.Contains(txt, want) {
			t.Errorf("чистый отчёт без %q:\n%s", want, txt)
		}
	}
	if strings.Contains(txt, "Упущенная") {
		t.Errorf("при нулевом долге и без нулевых актов блока быть не должно:\n%s", txt)
	}
}

func TestReportTextOnlyZeroActs(t *testing.T) {
	// всё оплачено, но есть неоценённый акт — выгода всё равно уходит
	st := store.Stats{Objects: 1, Acts: 2, Total: 5000, Paid: 5000, ZeroActs: 1}
	txt := reportText(st, 0)
	if !strings.Contains(txt, "Упущенная выгода") || !strings.Contains(txt, "1 акт с суммой 0") {
		t.Errorf("нет предупреждения о неоценённом акте:\n%s", txt)
	}
	if strings.Contains(txt, "не оплачено") {
		t.Errorf("долга нет — строки про оплату быть не должно:\n%s", txt)
	}
}

func TestReportTextPluralActs(t *testing.T) {
	cases := map[int]string{1: "1 акт", 2: "2 акта", 5: "5 актов", 11: "11 актов", 21: "21 акт"}
	for n, want := range cases {
		st := store.Stats{Total: 100, UnpaidActs: n}
		txt := reportText(st, 0)
		if !strings.Contains(txt, want) {
			t.Errorf("n=%d: хочу %q, получено:\n%s", n, want, txt)
		}
	}
}

func TestPhotoCaption(t *testing.T) {
	at := time.Date(2026, 8, 31, 15, 28, 0, 0, time.UTC)
	t.Run("полный", func(t *testing.T) {
		aid := int64(2)
		p := domain.PhotoRec{ActNo: 2, ActID: &aid, CreatedAt: at, Caption: "электрика, до штукатурки"}
		got := photoCaption(p, 3, 5)
		for _, want := range []string{"3/5", "31.08 15:28", "акт №2", "электрика, до штукатурки"} {
			if !strings.Contains(got, want) {
				t.Errorf("подпись без %q: %s", want, got)
			}
		}
	})
	t.Run("без акта и подписи", func(t *testing.T) {
		p := domain.PhotoRec{CreatedAt: at}
		got := photoCaption(p, 1, 1)
		if strings.Contains(got, "акт") || strings.Contains(got, "📝") {
			t.Errorf("лишнее в подписи: %s", got)
		}
		if !strings.Contains(got, "1/1") {
			t.Errorf("нет нумерации: %s", got)
		}
	})
}

func TestPhotoBtnTextTrim(t *testing.T) {
	long := strings.Repeat("Ж", 100) // 200 байт UTF-8
	got := photoBtnText(long, 3)
	if len(got) > 64 {
		t.Errorf("кнопка длиннее 64 байт (лимит Telegram): %d %q", len(got), got)
	}
	if !strings.Contains(got, "3 фото") {
		t.Errorf("нет счётчика: %q", got)
	}
	short := photoBtnText("ЖК Сонячний", 12)
	if !strings.Contains(short, "ЖК Сонячний — 12 фото") {
		t.Errorf("короткое имя обрезано зря: %q", short)
	}
}
