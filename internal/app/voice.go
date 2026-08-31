// Голосовой ввод позиций акта (v0.2): OGG → шлюз → текст → позиции → подтверждение.
package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"proakt/internal/domain"
	"proakt/internal/parse"
	"proakt/internal/tg"
)

// лимит голосового сообщения
const voiceMaxBytes = 10 << 20

// onVoiceMessage — получили голосовое.
func (b *Bot) onVoiceMessage(ctx context.Context, m tg.Message) {
	chatID := m.Chat.ID
	if b.ai == nil {
		b.text(ctx, chatID, "Голос пока не подключён — вводи позиции текстом 🙂")
		return
	}
	state, data, _ := b.st.State(ctx, chatID)
	if state != stActLines {
		b.textKB(ctx, chatID, "Голос удобно диктовать при вводе позиций акта — жми 📋 Новый акт.", MainMenu())
		return
	}
	if m.Voice.FileSize > voiceMaxBytes {
		b.text(ctx, chatID, "Голосовое длинновато 🎤 Диктуй, пожалуйста, порциями до ~5 минут.")
		return
	}

	_ = b.tg.SendChatAction(ctx, chatID)
	ogg, err := b.tg.DownloadFileBytes(ctx, m.Voice.FileID, voiceMaxBytes)
	if err != nil {
		log.Printf("голос: скачивание чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не смог скачать голосовое 😕 Попробуй ещё раз или введи текстом.")
		return
	}
	transcript, err := b.ai.Transcribe(ctx, ogg)
	if err != nil {
		log.Printf("голос: распознавание чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не получилось расшифровать 🎧\nПопробуй ещё раз или введи текстом.")
		return
	}
	if strings.TrimSpace(transcript) == "" {
		b.text(ctx, chatID, "Распознал пустой текст 🤔 Диктуй чуть громче или введи текстом.")
		return
	}
	b.text(ctx, chatID, "Распознал:\n"+transcript)

	// позиции: сначала модель (понимает числа словами), при сбое — локальный парсер
	lines, err := b.ai.ExtractPositions(ctx, transcript)
	if err != nil {
		log.Printf("голос: позиции чат %d: %v", chatID, err)
		lines = localParseLines(transcript)
	}
	// v0.3.6: единицы к канону («м2» → «м²»), как в текстовом вводе
	for i := range lines {
		if u := parse.NormalizeUnit(lines[i].Unit); u != "" {
			lines[i].Unit = u
		}
	}
	if len(lines) == 0 {
		b.text(ctx, chatID, "Не выделил позиции 🤔 Продиктуй медленнее или введи текстом:\nштукатурка 45 м² 260")
		return
	}
	if len(lines) > 200 {
		lines = lines[:200]
	}

	// v0.3: позиции без цены — дополняем из прайса, если он есть
	filledNote := ""
	items, catErr := b.st.ListCatalog(ctx, chatID)
	if catErr == nil && len(items) > 0 {
		var missing []string
		lines, missing = fillFromCatalog(items, lines)
		if n := len(missing); n > 0 {
			sh := missing
			if len(sh) > 3 {
				sh = sh[:3]
			}
			filledNote = fmt.Sprintf("\n\n⚠ Без цены (нет в прайсе): %s — продиктуй с ценой или добавь в 💵 Прайс", strings.Join(sh, ", "))
		}
		if len(lines) == 0 {
			b.text(ctx, chatID, "Цены не нашёл в прайсе 🤔\nПродиктуй с ценой: «штукатурка сорок пять метров двести шестьдесят»"+filledNote)
			return
		}
	} else if n, _ := zeroPriceLines(lines); n > 0 {
		// v0.3.6: прайс пуст — говорим прямо, а не молчим (иначе акт выйдет на 0 грн)
		filledNote = "\n\n⚠ Позиции без цены: прайс пуст. Загрузи его (💵 Прайс → 📥 Импорт из файла) или диктуй с ценой."
	}

	data["voice"] = linesJSON(lines)
	_ = b.st.SetState(ctx, chatID, stActLines, data)

	var sb strings.Builder
	sb.WriteString("Проверь, что распознал верно:\n\n")
	for i, l := range lines {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, describeLine(l)))
	}
	sb.WriteString(fmt.Sprintf("\nПозиций: %d · сумма: %s", len(lines), money(sumDraft(lines))))
	sb.WriteString(filledNote)
	_ = b.tg.SendMessage(ctx, chatID, sb.String(), tg.Inline(tg.KB{
		{tg.KBButton{Text: "✅ Добавить в акт", CallbackData: "vok"},
			tg.KBButton{Text: "❌ Отбросить", CallbackData: "vox"}},
	}))
}

// applyVoiceLines — «✅ Добавить в акт».
func (b *Bot) applyVoiceLines(ctx context.Context, chatID int64, data map[string]string) {
	voice := parseDraft(data["voice"])
	if len(voice) == 0 {
		b.text(ctx, chatID, "Голосовые позиции устарели — надиктуй заново 🎤")
		return
	}
	delete(data, "voice")
	lines := append(parseDraft(data["lines"]), voice...)
	if len(lines) > 200 {
		lines = lines[:200]
	}
	data["lines"] = linesJSON(lines)
	_ = b.st.SetState(ctx, chatID, stActLines, data)
	_ = b.tg.SendChatAction(ctx, chatID)
	b.text(ctx, chatID, fmt.Sprintf(
		"✅ Добавил позиций: %d\n—\nВсего в акте: %d · сумма: %s\nМожно диктовать дальше, вводить текстом или завершить.",
		len(voice), len(lines), money(sumDraft(lines))))
}

// discardVoiceLines — «❌ Отбросить».
func (b *Bot) discardVoiceLines(ctx context.Context, chatID int64, data map[string]string) {
	delete(data, "voice")
	_ = b.st.SetState(ctx, chatID, stActLines, data)
	b.text(ctx, chatID, "Отбросил 🗑 Продиктуй заново или введи текстом.")
}

// localParseLines — запасной локальный разбор (если модель недоступна).
func localParseLines(s string) []domain.DraftLine {
	var lines []domain.DraftLine
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if l, ok := parse.ParsePosition(ln); ok {
			lines = append(lines, l)
		}
	}
	return lines
}
