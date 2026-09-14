// Автобэкап смет и актов (v0.6). ТЗ: «авто-бэкап смет/актов по расписанию».
// Раз в сутки каждый чат выгружается в JSON (gzip) в FILES_DIR/backups/ —
// файлы забирает обычный бэкап-агент владельца (rsync/rclone в облако).
// Плюс /backup — прислать свежую выгрузку прямо в чат одним файлом.
package app

import (
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const backupInterval = 24 * time.Hour

// sendBackup — /backup: выгрузка чата файлом в чат.
func (b *Bot) sendBackup(ctx context.Context, chatID int64) {
	_ = b.tg.SendChatAction(ctx, chatID)
	path, n, err := b.writeChatBackup(ctx, chatID)
	if err != nil {
		log.Printf("бэкап чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не собрал выгрузку 😕 Попробуй ещё раз.")
		return
	}
	caption := fmt.Sprintf("💾 Бэкап: сметы, акты, прайс, шаблоны (%d записей).\nJSON + gzip — пригодится при переезде или для истории.", n)
	if err := b.tg.SendDocumentFile(ctx, chatID, path, caption); err != nil {
		b.text(ctx, chatID, "Файл не отправился 😕 но лежит на сервере: "+path)
	}
}

// writeChatBackup — собрать JSON и записать (gzip) в backups/.
// Возвращает путь и число сохранённых сущностей.
func (b *Bot) writeChatBackup(ctx context.Context, chatID int64) (string, int, error) {
	dump, err := b.st.BackupChatDump(ctx, chatID)
	if err != nil {
		return "", 0, err
	}
	raw, err := dump.Marshal()
	if err != nil {
		return "", 0, err
	}
	dir := filepath.Join(b.files, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, err
	}
	name := fmt.Sprintf("chat_%d_%s.json.gz", chatID, time.Now().Format("2006-01-02"))
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	zw := gzip.NewWriter(f)
	if _, err := zw.Write(raw); err != nil {
		return "", 0, err
	}
	if err := zw.Close(); err != nil {
		return "", 0, err
	}
	n := len(dump.Estimates) + len(dump.Acts) + len(dump.Price) + len(dump.Templates)
	return path, n, nil
}

// StartBackupLoop — фоновая выгрузка всех чатов раз в сутки (goroutine из main).
// Бэкап пишется «в tyль» всем чатам с данными (объекты/сметы/акты).
func (b *Bot) StartBackupLoop(ctx context.Context) {
	go func() {
		t := time.NewTicker(backupInterval)
		defer t.Stop()
		// первый бэкап — через час после старта (не мешаем старту)
		first := time.AfterFunc(time.Hour, func() { b.backupAll(ctx) })
		defer first.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				b.backupAll(ctx)
			}
		}
	}()
}

// backupAll — выгрузка всех известных чатов (у которых есть хоть что-то).
func (b *Bot) backupAll(ctx context.Context) {
	chats, err := b.st.BackupChatIDs(ctx)
	if err != nil {
		log.Printf("бэкап: список чатов: %v", err)
		return
	}
	for _, chatID := range chats {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if _, _, err := b.writeChatBackup(ctx, chatID); err != nil {
			log.Printf("бэкап чат %d: %v", chatID, err)
			continue
		}
		log.Printf("бэкап: чат %d выгружен", chatID)
	}
	// подчищаем старые бэкапы — диск на VM не резиновый
	b.pruneBackups(30)
}

// pruneBackups — держим последние keepDays дней выгрузок.
func (b *Bot) pruneBackups(keepDays int) {
	dir := filepath.Join(b.files, "backups")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	deadline := time.Now().AddDate(0, 0, -keepDays)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.IsDir() {
			continue
		}
		if info.ModTime().Before(deadline) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}
