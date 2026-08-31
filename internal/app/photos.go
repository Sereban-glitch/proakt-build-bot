package app

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"proakt/internal/domain"
	"proakt/internal/tg"
)

// startPhotoWait — «📷 Фото»: ждём снимок.
func (b *Bot) startPhotoWait(ctx context.Context, chatID int64) {
	objs, err := b.st.ListObjects(ctx, chatID)
	if err != nil || len(objs) == 0 {
		b.textKB(ctx, chatID, "Фото крепятся к объекту — сначала создадим его.", CancelMenu())
		b.beginObject(ctx, chatID, map[string]string{})
		return
	}
	_ = b.st.SetState(ctx, chatID, stPhotoWait, map[string]string{})
	b.textKB(ctx, chatID, "Пришли фото скрытых работ 📷\nПодпись к фото станет комментарием (например: «электрика, до штукатурки»).", CancelMenu())
}

// onPhotoMessage — получили фото (в любом «спокойном» состоянии).
func (b *Bot) onPhotoMessage(ctx context.Context, m tg.Message) {
	chatID := m.Chat.ID
	state, _, _ := b.st.State(ctx, chatID)
	switch state {
	case stIdle, stPhotoWait:
	default:
		b.text(ctx, chatID, "Сейчас другое действие в процессе — заверши его или ⏹ Отмена, потом пришли фото.")
		return
	}

	objs, err := b.st.ListObjects(ctx, chatID)
	if err != nil || len(objs) == 0 {
		b.textKB(ctx, chatID, "Фото крепятся к объекту — сначала создай объект: /objects", CancelMenu())
		return
	}

	// выбираем самый большой размер
	var best tg.PhotoSize
	for _, p := range m.Photo {
		if p.Width*p.Height >= best.Width*best.Height {
			best = p
		}
	}

	acts, _ := b.st.ListActs(ctx, chatID, 6)
	rows := []tg.KBButton{}
	for _, a := range acts {
		rows = append(rows, tg.KBButton{
			Text:         fmt.Sprintf("📋 Акт №%d · %s", a.ActNo, a.ObjectName),
			CallbackData: fmt.Sprintf("pha:%d", a.ID),
		})
	}
	for _, o := range objs {
		rows = append(rows, tg.KBButton{
			Text:         "🏠 " + o.Name,
			CallbackData: fmt.Sprintf("pho:%d", o.ID),
		})
	}
	rows = append(rows, tg.KBButton{Text: BtnCancel, CallbackData: "cancel"})

	data := map[string]string{"file_id": best.FileID, "caption": strings.TrimSpace(m.Caption)}
	_ = b.st.SetState(ctx, chatID, stPhotoPick, data)
	b.textKB(ctx, chatID, "📷 Фото получил. Куда крепим?", tg.Inline(tg.KB{rows}))
}

// attachPhoto — привязка и загрузка фото (по выбору кнопки).
func (b *Bot) attachPhoto(ctx context.Context, chatID int64, data map[string]string, actID *int64, objectID int64) {
	fileID := data["file_id"]
	if fileID == "" {
		b.reset(ctx, chatID)
		b.text(ctx, chatID, "Фото потерялось — пришли заново 📷")
		return
	}

	obj := int64(0)
	label := ""
	if actID != nil && *actID > 0 {
		brief, err := b.st.GetAct(ctx, *actID)
		if err != nil {
			b.reset(ctx, chatID)
			b.text(ctx, chatID, "Акт не найден 😕")
			return
		}
		obj = brief.ObjectID
		label = fmt.Sprintf("act_%d", brief.ActNo)
	} else if objectID > 0 {
		o, err := b.st.GetObject(ctx, objectID)
		if err != nil {
			b.reset(ctx, chatID)
			b.text(ctx, chatID, "Объект не найден 😕")
			return
		}
		obj = o.ID
	} else {
		b.reset(ctx, chatID)
		b.text(ctx, chatID, "Не понял, куда крепить фото 🤔")
		return
	}

	_ = b.tg.SendChatAction(ctx, chatID)
	name := fmt.Sprintf("%s_%d.jpg", label, time.Now().Unix())
	if label == "" {
		name = fmt.Sprintf("obj_%d", time.Now().Unix())
	}
	path := filepath.Join(b.files, "photos", fmt.Sprintf("obj_%d", obj), name)
	if err := b.tg.DownloadFile(ctx, fileID, path); err != nil {
		log.Printf("download фото чат %d: %v", chatID, err)
		// даже без файла — сохраним запись (file_id в Telegram живёт)
		path = ""
	}
	if err := b.st.AddPhoto(ctx, actID, obj, fileID, path, data["caption"]); err != nil {
		b.reset(ctx, chatID)
		b.text(ctx, chatID, "Не записал фото в базу 😕")
		return
	}
	b.reset(ctx, chatID)
	caption := data["caption"]
	if caption != "" {
		caption = "\n📝 " + caption
	}
	b.textKB(ctx, chatID, "📷 Фото сохранено"+whereLabel(actID)+caption, MainMenu())
}

// --- показ сохранённых фото (v0.3.7) -------------------------------------------
// Инсайт из видео: скрытые работы нужно не только сохранить, но и показать —
// заказчику при сдаче, себе при споре. Фото уже в Telegram: отправляем по file_id.

// showPhotos — выслать фото скрытых работ объекта (actID > 0: только фото акта).
func (b *Bot) showPhotos(ctx context.Context, chatID int64, objectID, actID int64) {
	obj, err := b.st.GetObject(ctx, objectID)
	if err != nil {
		b.text(ctx, chatID, "Объект не найден 😕 Список: /objects")
		return
	}
	phs, err := b.st.ListPhotos(ctx, objectID, actID)
	if err != nil {
		b.text(ctx, chatID, "Не смог прочитать фото 😕")
		return
	}
	if len(phs) == 0 {
		scope := "объекта"
		if actID > 0 {
			scope = "акта"
		}
		b.textKB(ctx, chatID, fmt.Sprintf(
			"📷 Фото %s «%s» пока нет.\nПришли снимок скрытых работ (например: «электрика, до штукатурки») — сохраню и покажу здесь же.", scope, obj.Name), MainMenu())
		return
	}

	// сначала сводка, потом сами снимки — чтобы чат читался как фотоотчёт
	header := fmt.Sprintf("📷 Скрытые работы · «%s» — %d фото:", obj.Name, len(phs))
	if actID > 0 {
		header = fmt.Sprintf("📷 Фото по акту · «%s» — %d:", obj.Name, len(phs))
	}
	b.text(ctx, chatID, header)

	sent := 0
	for i := len(phs) - 1; i >= 0; i-- { // старые → новые, как хронология стройки
		p := phs[i]
		cap := photoCaption(p, len(phs)-i, len(phs))
		if err := b.tg.SendPhoto(ctx, chatID, p.FileID, cap); err != nil {
			log.Printf("sendPhoto чат %d фото %d: %v", chatID, p.ID, err)
			continue
		}
		sent++
	}
	if sent < len(phs) {
		b.textKB(ctx, chatID, fmt.Sprintf("Показал %d из %d — остальные, увы, устарели в Telegram 😕", sent, len(phs)), MainMenu())
		return
	}
	b.textKB(ctx, chatID, "Вот всё, что сохранил 📷 Фото — всегда под рукой: /objects", MainMenu())
}

// photoCaption — подпись к фото при показе: №, дата, акт, комментарий мастера.
func photoCaption(p domain.PhotoRec, n, total int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "📷 %d/%d · %s", n, total, p.CreatedAt.Format("02.01 15:04"))
	if p.ActNo > 0 {
		fmt.Fprintf(&sb, " · акт №%d", p.ActNo)
	}
	if c := strings.TrimSpace(p.Caption); c != "" {
		sb.WriteString("\n📝 " + c)
	}
	return sb.String()
}

func whereLabel(actID *int64) string {
	if actID != nil && *actID > 0 {
		return " (привязано к акту)"
	}
	return " (к объекту)"
}
