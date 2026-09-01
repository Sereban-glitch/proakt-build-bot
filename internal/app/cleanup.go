// cleanup.go — «навигация в обе стороны» (v0.3.8).
// Всё, что можно добавить, можно и убрать: оплаты, объекты (фото — в photos.go,
// акты — в acts.go, позиции прайса — в price.go, строки акта — ↩️ Убрать последнюю).
// «Работало как часики: просто и удобно» — поэтому у каждого действия кнопка,
// подтверждение и понятный отчёт, что именно убрано.
package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"proakt/internal/domain"
	"proakt/internal/tg"
)

// --- оплаты: список и удаление ошибочной (v0.3.8) ------------------------------

// showPayments — «💰 оплаты №…»: сколько и когда пришло по акту.
// Ошибочная оплата (лишний ноль и т.п.) убирается кнопкой — долг
// в отчёте тут же возвращается к правде.
func (b *Bot) showPayments(ctx context.Context, chatID int64, actID int64) {
	brief, err := b.st.GetAct(ctx, actID)
	if err != nil {
		b.text(ctx, chatID, "Акт не найден 😕 Список: /acts")
		return
	}
	pays, err := b.st.Payments(ctx, actID)
	if err != nil {
		b.text(ctx, chatID, "Не смог прочитать оплаты 😕")
		return
	}
	if len(pays) == 0 {
		b.textKB(ctx, chatID, fmt.Sprintf(
			"Акт №%d · %s\nОплат пока нет — отметить: 💰 Долги", brief.ActNo, brief.ObjectName), MainMenu())
		return
	}
	rows := []tg.KBButton{}
	for _, p := range pays {
		rows = append(rows, tg.KBButton{
			Text:         fmt.Sprintf("🗑 %s · от %s", money(p.Amount), p.At.Format("02.01")),
			CallbackData: fmt.Sprintf("paydel:%d", p.ID),
		})
	}
	b.textKB(ctx, chatID, paymentListText(brief, pays), tg.Inline(tg.KB{rows}))
}

// paymentListText — сводка оплат акта (вынесено для тестов).
func paymentListText(b domain.ActBrief, pays []domain.Payment) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "💰 Оплаты акта №%d · «%s»:\n\n", b.ActNo, b.ObjectName)
	for _, p := range pays {
		fmt.Fprintf(&sb, "• %s — %s\n", p.At.Format("02.01"), money(p.Amount))
	}
	fmt.Fprintf(&sb, "\nСумма акта: %s\nОплачено: %s\n", money(b.Total), money(b.Paid))
	if b.Balance() > 0.009 {
		fmt.Fprintf(&sb, "Остаток: %s", money(b.Balance()))
	} else {
		sb.WriteString("Акт закрыт полностью ✅")
	}
	sb.WriteString("\n\n🗑 — убрать ошибочную оплату (долг по акту вырастет).")
	return sb.String()
}

// confirmDeletePayment — спросить перед удалением оплаты.
func (b *Bot) confirmDeletePayment(ctx context.Context, chatID int64, cqID string, payID int64) {
	p, err := b.st.GetPayment(ctx, chatID, payID)
	if err != nil {
		_ = b.tg.AnswerCallbackQuery(ctx, cqID, "Оплата не найдена")
		b.text(ctx, chatID, "Оплата уже удалена 😕 Списки: /acts")
		return
	}
	_ = b.tg.AnswerCallbackQuery(ctx, cqID, "")
	b.textKB(ctx, chatID, paymentDelText(p), tg.Inline(tg.KB{
		{tg.KBButton{Text: "🗑 Удалить", CallbackData: fmt.Sprintf("paydelyes:%d", p.ID)}},
		{tg.KBButton{Text: "❌ Оставить", CallbackData: "paydelno"}},
	}))
}

// paymentDelText — подтверждение удаления оплаты (вынесено для тестов).
func paymentDelText(p domain.PaymentRec) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "🗑 Удалить оплату %s от %s?\nАкт №%d · «%s»",
		money(p.Amount), p.At.Format("02.01"), p.ActNo, p.ObjName)
	sb.WriteString("\n\nДолг по акту вырастет на эту сумму.\nЗаписать заново всегда можно: 💰 Долги")
	return sb.String()
}

// deletePaymentDo — само удаление + новый остаток по акту.
func (b *Bot) deletePaymentDo(ctx context.Context, chatID int64, cqID string, payID int64) {
	p, err := b.st.DeletePayment(ctx, chatID, payID)
	if err != nil {
		_ = b.tg.AnswerCallbackQuery(ctx, cqID, "Не получилось")
		b.textKB(ctx, chatID, "Не получилось удалить оплату 😕 /acts", MainMenu())
		return
	}
	_ = b.tg.AnswerCallbackQuery(ctx, cqID, "Удалил")
	var tail string
	if brief, err := b.st.GetAct(ctx, p.ActID); err == nil {
		tail = paymentDeletedText(brief)
	}
	b.textKB(ctx, chatID, fmt.Sprintf("🗑 Оплата %s удалена.\n%s", money(p.Amount), tail), MainMenu())
}

// paymentDeletedText — отчёт после удаления: сколько теперь по акту.
func paymentDeletedText(b domain.ActBrief) string {
	if b.Balance() > 0.009 {
		return fmt.Sprintf("Акт №%d: оплачено %s из %s, остаток %s.\nОтметить приход: 💰 Долги",
			b.ActNo, money(b.Paid), money(b.Total), money(b.Balance()))
	}
	return fmt.Sprintf("Акт №%d: оплачено %s из %s.", b.ActNo, money(b.Paid), money(b.Total))
}

// --- объекты: скрыть лишний (v0.3.8) -------------------------------------------

// pickObjectToHide — «🗑 Убрать объект»: выбрать какой.
func (b *Bot) pickObjectToHide(ctx context.Context, chatID int64, cqID string) {
	objs, err := b.st.ListObjects(ctx, chatID)
	if err != nil || len(objs) == 0 {
		_ = b.tg.AnswerCallbackQuery(ctx, cqID, "Объектов нет")
		b.text(ctx, chatID, "Объектов пока нет 🏠")
		return
	}
	_ = b.tg.AnswerCallbackQuery(ctx, cqID, "")
	rows := []tg.KBButton{}
	for _, o := range objs {
		rows = append(rows, tg.KBButton{
			Text:         "🗑 " + btnTrim(o.Name),
			CallbackData: fmt.Sprintf("objdel:%d", o.ID),
		})
	}
	b.textKB(ctx, chatID, "Какой объект убрать из меню?\n(акты, оплаты и фото по нему останутся в /acts и /report)",
		tg.Inline(tg.KB{rows}))
}

// btnTrim — имя объекта в кнопку (лимит Telegram 64 байта, бьём по рунам).
func btnTrim(name string) string {
	const budget = 40 // 64 − «🗑 »(5) − запас
	label := name
	for len(label) > budget {
		r := []rune(label)
		label = string(r[:len(r)-1])
	}
	return label
}

// confirmHideObject — спросить перед скрытием объекта.
func (b *Bot) confirmHideObject(ctx context.Context, chatID int64, cqID string, objID int64) {
	o, err := b.st.GetObject(ctx, objID)
	if err != nil {
		_ = b.tg.AnswerCallbackQuery(ctx, cqID, "Объект не найден")
		b.text(ctx, chatID, "Объект не найден 😕 /objects")
		return
	}
	_ = b.tg.AnswerCallbackQuery(ctx, cqID, "")
	b.textKB(ctx, chatID, fmt.Sprintf(
		"🗑 Убрать «%s» из меню?\n\nОбъект исчезнет из списков выбора.\nАкты, оплаты и фото останутся на месте: /acts и /report.\n\nСделал объект по ошибке — самое то.", o.Name),
		tg.Inline(tg.KB{
			{tg.KBButton{Text: "🗑 Убрать из меню", CallbackData: fmt.Sprintf("objdelyes:%d", o.ID)}},
			{tg.KBButton{Text: "❌ Оставить", CallbackData: "objdelno"}},
		}))
}

// hideObjectDo — само скрытие (status='archived', данные целы).
func (b *Bot) hideObjectDo(ctx context.Context, chatID int64, cqID string, objID int64) {
	name, err := b.st.ArchiveObject(ctx, chatID, objID)
	if err != nil {
		_ = b.tg.AnswerCallbackQuery(ctx, cqID, "Не получилось")
		b.textKB(ctx, chatID, "Не получилось убрать объект 😕 /objects", MainMenu())
		return
	}
	_ = b.tg.AnswerCallbackQuery(ctx, cqID, "Убрал")
	b.textKB(ctx, chatID, fmt.Sprintf(
		"🗑 Объект «%s» убран из меню.\nЕго акты и деньги целы: /acts · /report", name), MainMenu())
}

// cbID — защита от опечаток: парсер id из callback-данных.
func cbID(data string, prefix string) int64 {
	id, _ := strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
	return id
}
