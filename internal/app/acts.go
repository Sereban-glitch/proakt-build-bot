// Список актов и удаление ошибочного акта (v0.3.6).
// «Линза строителя»: ошибочный акт не должен висеть в отчётах —
// /acts → 🗑 → подтверждение → акт, позиции и оплаты уходят, фото остаётся у объекта.
package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"proakt/internal/tg"
)

// showActs — список актов с кнопками удаления и оплат (v0.3.8: «💰 оплаты» —
// сколько и когда пришло, ошибочную оплату можно убрать прямо оттуда).
func (b *Bot) showActs(ctx context.Context, chatID int64) {
	acts, err := b.st.ListActs(ctx, chatID, 12)
	if err != nil || len(acts) == 0 {
		b.textKB(ctx, chatID, "Актов пока нет. Создай первый: 📋 Новый акт", MainMenu())
		return
	}
	var sb strings.Builder
	sb.WriteString("📋 Последние акты:\n\n")
	rows := []tg.KBButton{}
	photoRows := tg.KB{} // v0.3.7: фото акта можно показать одной кнопкой
	payRows := tg.KB{}   // v0.3.8: оплаты акта списком
	for _, a := range acts {
		status := "✅ оплачен"
		if a.Balance() > 0.009 {
			status = "долг " + money(a.Balance())
		}
		line := fmt.Sprintf("• №%d (%s) — %s · %s", a.ActNo, a.ObjectName, money(a.Total), status)
		if a.Paid > 0.009 && a.Balance() > 0.009 {
			line += fmt.Sprintf(" · оплачено %s", money(a.Paid))
		}
		if a.Photos > 0 {
			line += fmt.Sprintf(" · 📷 %d", a.Photos)
			photoRows = append(photoRows, []tg.KBButton{{
				Text:         fmt.Sprintf("📷 фото акта №%d", a.ActNo),
				CallbackData: fmt.Sprintf("pav:%d", a.ID),
			}})
		}
		if a.Paid > 0.009 { // v0.3.8: оплаченное — видно и редактируется
			payRows = append(payRows, []tg.KBButton{{
				Text:         fmt.Sprintf("💰 оплаты №%d", a.ActNo),
				CallbackData: fmt.Sprintf("plist:%d", a.ID),
			}})
		}
		sb.WriteString(line + "\n")
		rows = append(rows, tg.KBButton{
			Text:         fmt.Sprintf("🗑 №%d · %s · %s", a.ActNo, a.ObjectName, money(a.Total)),
			CallbackData: fmt.Sprintf("adel:%d", a.ID),
		})
	}
	kb := tg.KB{rows}
	kb = append(photoRows, kb...)
	kb = append(payRows, kb...)
	sb.WriteString("\n💰 — оплаты акта · 📷 — фото · 🗑 — удалить ошибочный акт (с подтверждением)")
	b.textKB(ctx, chatID, sb.String(), tg.Inline(kb))
}

// confirmDeleteAct — «🗑 №N»: показать, что уйдёт, и спросить.
func (b *Bot) confirmDeleteAct(ctx context.Context, chatID int64, cqID string, actID int64) {
	brief, err := b.st.GetAct(ctx, actID)
	if err != nil {
		_ = b.tg.AnswerCallbackQuery(ctx, cqID, "Акт не найден")
		b.text(ctx, chatID, "Акт не найден 😕 Список: /acts")
		return
	}
	_ = b.tg.AnswerCallbackQuery(ctx, cqID, "")
	_ = b.st.SetState(ctx, chatID, stActDelConf, map[string]string{"act_id": fmt.Sprint(actID)})
	warn := ""
	if brief.Paid > 0.009 {
		warn = fmt.Sprintf("\n⚠️ Вместе с оплатами на %s!", money(brief.Paid))
	}
	b.textKB(ctx, chatID, fmt.Sprintf(
		"🗑 Удалить акт №%d (%s) на %s?\n\nУйдут: позиции и оплаты.\nОстанутся: фото (привяжутся к объекту) и XLSX, уже отправленный в чат.%s\n\nВернуть будет нельзя.",
		brief.ActNo, brief.ObjectName, money(brief.Total), warn), tg.Inline(tg.KB{
		{tg.KBButton{Text: "🗑 Удалить", CallbackData: "adelyes"}},
		{tg.KBButton{Text: "❌ Оставить", CallbackData: "adelno"}},
	}))
}

// deleteActDo — «🗑 Удалить»: удалить и отчитаться.
func (b *Bot) deleteActDo(ctx context.Context, chatID int64) {
	state, data, _ := b.st.State(ctx, chatID)
	if state != stActDelConf {
		b.textKB(ctx, chatID, "Уже неактуально. Акты: /acts", MainMenu())
		return
	}
	actID, _ := strconv.ParseInt(data["act_id"], 10, 64)
	brief, err := b.st.DeleteAct(ctx, actID)
	b.reset(ctx, chatID)
	if err != nil {
		b.textKB(ctx, chatID, "Не получилось удалить 😕", MainMenu())
		return
	}
	b.textKB(ctx, chatID, fmt.Sprintf(
		"🗑 Акт №%d (%s) удалён.\nСводка: /report · акты: /acts", brief.ActNo, brief.ObjectName), MainMenu())
}
