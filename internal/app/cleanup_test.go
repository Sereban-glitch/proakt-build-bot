// Тесты v0.3.8 «навигация в обе стороны»: инструкция по шагам, удаление
// фото/оплат, скрытие объекта. Владелец: «всё должно работать как часики —
// добавить И удалить, просто и удобно, на языке строителя».
package app

import (
	"strings"
	"testing"
	"time"

	"proakt/internal/ai"
	"proakt/internal/domain"
)

// --- инструкция и приветствие --------------------------------------------------

func TestInstructionStepsAndCleanup(t *testing.T) {
	b := &Bot{ai: nil}
	txt := b.instruction()
	// 7 шагов — полный маршрут мастера, ничего не пропущено
	for _, want := range []string{
		"ИНСТРУКЦИЯ", "ШАГ 1", "ШАГ 2", "ШАГ 3", "ШАГ 4", "ШАГ 5", "ШАГ 6", "ШАГ 7",
		"Импорт из файла", "Новый объект", "Новый акт", "Завершить акт",
		"Долги", "СКРЫТЫЕ РАБОТЫ", "Отчёт",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("инструкция без %q", want)
		}
	}
	// «в обе стороны»: раздел удаления покрывает ВСЕ сущности
	for _, want := range []string{
		"УДАЛИТЬ МОЖНО ВСЁ",
		"↩️ Убрать последнюю",   // позиция акта
		"/acts → 🗑",             // акт
		"неудачное фото",        // фото
		"ошибочную оплату",      // оплата
		"позицию прайса",        // прайс
		"лишний объект",         // объект
		"ПРИВЕТ"[:0] + "формат", // масштабируемость (форматы/таблицы)
		"PDF", "дополнительные таблицы",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("инструкция без %q", want)
		}
	}
	// без голосового шлюза — нет обещания голоса
	if strings.Contains(txt, "голосом") {
		t.Errorf("ai выключен, а инструкция обещает голос:\n%s", txt)
	}
}

func TestInstructionVoiceOn(t *testing.T) {
	b := &Bot{ai: &ai.Gateway{}} // нулевой шлюз — но факт «голос включён» есть
	txt := b.instruction()
	if !strings.Contains(txt, "голосом 🎤") {
		t.Errorf("при включённом голосе нет подсказки про диктовку")
	}
}

// Лимит Telegram: 4096 символов на сообщение (запас — 96).
func TestTextsFitTelegram(t *testing.T) {
	b := &Bot{ai: &ai.Gateway{}}
	for name, txt := range map[string]string{
		"instruction": b.instruction(),
		"welcome":     b.welcome(),
		"help":        b.help(),
	} {
		if n := len([]rune(txt)); n > 4000 {
			t.Errorf("%s: %d рун — больше лимита Telegram", name, n)
		} else {
			t.Logf("%s: %d рун — ок", name, n)
		}
	}
}

func TestWelcomePointsToInstrButton(t *testing.T) {
	b := &Bot{ai: &ai.Gateway{}}
	w := b.welcome()
	for _, want := range []string{"кнопк", "👇", "Что это даёт", "ДИКТОВАТЬ голосом"} {
		if !strings.Contains(w, want) {
			t.Errorf("приветствие без %q:\n%s", want, w)
		}
	}
	// кнопка инструкции — валидный callback
	kb := InstrButton()
	if len(kb.InlineKeyboard) != 1 || kb.InlineKeyboard[0][0].CallbackData != "instr" {
		t.Errorf("кнопка инструкции не собралась: %+v", kb)
	}
}

// --- удаление фото -------------------------------------------------------------

func TestPhotoDelText(t *testing.T) {
	at := time.Date(2026, 9, 1, 10, 30, 0, 0, time.UTC)
	aid := int64(3)
	t.Run("полное", func(t *testing.T) {
		p := domain.PhotoRec{ID: 7, ActNo: 3, ActID: &aid, CreatedAt: at, Caption: "штроба до заделки"}
		got := photoDelText(p)
		for _, want := range []string{"01.09 10:30", "акт №3", "штроба до заделки", "прислать заново"} {
			if !strings.Contains(got, want) {
				t.Errorf("подтверждение без %q: %s", want, got)
			}
		}
	})
	t.Run("минимум", func(t *testing.T) {
		got := photoDelText(domain.PhotoRec{CreatedAt: at})
		if strings.Contains(got, "акт №") || strings.Contains(got, "📝") {
			t.Errorf("лишнее в минимальном подтверждении: %s", got)
		}
	})
}

// --- оплаты: список и удаление ---------------------------------------------------

func TestPaymentListText(t *testing.T) {
	at := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	pays := []domain.Payment{
		{ID: 1, ActID: 3, Amount: 10000, At: at},
		{ID: 2, ActID: 3, Amount: 1700, At: at.AddDate(0, 0, 2)},
	}
	t.Run("акт закрыт", func(t *testing.T) {
		b := domain.ActBrief{ActNo: 1, ObjectName: "ЖК Сонячний, кв. 12", Total: 11700, Paid: 11700}
		got := paymentListText(b, pays)
		for _, want := range []string{"№1", "ЖК Сонячний, кв. 12", "05.08 — 10 000 грн", "07.08 — 1 700 грн", "закрыт полностью ✅", "убрать ошибочную оплату"} {
			if !strings.Contains(got, want) {
				t.Errorf("список оплат без %q:\n%s", want, got)
			}
		}
	})
	t.Run("с долгом", func(t *testing.T) {
		b := domain.ActBrief{ActNo: 2, ObjectName: "ЖК Сонячний, кв. 12", Total: 11700, Paid: 10000}
		got := paymentListText(b, pays[:1])
		if !strings.Contains(got, "Остаток: 1 700 грн") {
			t.Errorf("нет остатка в списке оплат:\n%s", got)
		}
	})
}

func TestPaymentDelText(t *testing.T) {
	at := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	p := domain.PaymentRec{ID: 1, ActID: 3, Amount: 10000, At: at, ActNo: 3, ObjName: "ЖК Сонячний, кв. 45"}
	got := paymentDelText(p)
	for _, want := range []string{"10 000 грн", "от 05.08", "Акт №3", "ЖК Сонячний, кв. 45", "Долги"} {
		if !strings.Contains(got, want) {
			t.Errorf("подтверждение без %q: %s", want, got)
		}
	}
}

func TestPaymentDeletedText(t *testing.T) {
	b := domain.ActBrief{ActNo: 3, Total: 11700, Paid: 1700}
	if got := paymentDeletedText(b); !strings.Contains(got, "остаток 10 000 грн") {
		t.Errorf("нет остатка после удаления: %s", got)
	}
	b.Paid = 11700
	if got := paymentDeletedText(b); !strings.Contains(got, "оплачено 11 700 грн из 11 700 грн") {
		t.Errorf("закрытый акт после удаления: %s", got)
	}
}

// --- объекты: скрытие ------------------------------------------------------------

func TestBtnTrim(t *testing.T) {
	long := strings.Repeat("Ж", 100) // 200 байт
	got := "🗑 " + btnTrim(long)
	if len(got) > 64 {
		t.Errorf("кнопка длиннее 64 байт: %d", len(got))
	}
	if btnTrim("ЖК Сонячний, кв. 45") != "ЖК Сонячний, кв. 45" {
		t.Errorf("короткое имя обрезано зря")
	}
}

func TestCbID(t *testing.T) {
	if got := cbID("phdel:42", "phdel:"); got != 42 {
		t.Errorf("cbID = %d, хочу 42", got)
	}
	if got := cbID("phdel:abc", "phdel:"); got != 0 {
		t.Errorf("мусор в id должен давать 0, получил %d", got)
	}
}
