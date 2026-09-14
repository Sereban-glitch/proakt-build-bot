// Прайс из Google Таблиц (v0.6): ссылка → CSV-экспорт → мерж в прайс.
// Таблица = источник истины: мастер правит цены там, где удобно, —
// бот подтягивает их по кнопке 🔄 Синк, команде /sync и по расписанию.
package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"proakt/internal/domain"
	"proakt/internal/gsheet"
	"proakt/internal/store"
	"proakt/internal/tg"
)

// priceSheetSetup — настройка/просмотр источника прайса.
func (b *Bot) priceSheetSetup(ctx context.Context, chatID int64) {
	src, err := b.st.PriceSource(ctx, chatID)
	if err != nil {
		if err == store.ErrNotFound {
			b.beginPriceSheet(ctx, chatID)
			return
		}
		log.Printf("прайс: источник чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не прочитал настройку 😕 Попробуй ещё раз.")
		return
	}
	lastSync := "ещё не синхронизировалась"
	if !src.LastSync.IsZero() && src.LastSync.Year() > 1980 {
		lastSync = src.LastSync.Format("02.01 15:04")
		if src.LastError != "" {
			lastSync += " — ошибка: " + src.LastError
		} else {
			lastSync += fmt.Sprintf(" — записано позиций: %d", src.LastCount)
		}
	}
	b.textKB(ctx, chatID, fmt.Sprintf(
		"🌐 Google Таблица подключена:\n%s\n\nПоследний синк: %s\n\n🔄 Синк — подтянуть цены сейчас (таблица = источник истины:\nобновлю существующие, добавлю новые, ничего не удаляю).",
		src.URL, lastSync), priceSheetMenu(true))
}

// beginPriceSheet — просим ссылку.
func (b *Bot) beginPriceSheet(ctx context.Context, chatID int64) {
	_ = b.st.SetState(ctx, chatID, stPriceSheet, map[string]string{})
	b.textKB(ctx, chatID, `Пришли ссылку на Google Таблицу с прайсом:
https://docs.google.com/spreadsheets/d/…/edit#gid=0

Доступ: «все, у кого есть ссылка — Читатель» (или Файл → Опубликовать в сети).
Ключи Google не нужны — бот читает публичный CSV-экспорт.
Колонки: название, [единица], цена — порядок не важен.`, CancelMenu())
}

// priceSheetFromText — сохраняем ссылку и сразу синхронизируем.
func (b *Bot) priceSheetFromText(ctx context.Context, chatID int64, text string) {
	fileID, gid, err := gsheet.ParseSheetURL(text)
	if err != nil {
		b.text(ctx, chatID, "🤔 "+err.Error())
		return
	}
	if err := b.st.SetPriceSource(ctx, chatID, strings.TrimSpace(text), fileID, gid); err != nil {
		b.text(ctx, chatID, "Не сохранил ссылку 😕 Попробуй ещё раз.")
		return
	}
	b.reset(ctx, chatID)
	b.text(ctx, chatID, "✅ Ссылку сохранил. Синхронизирую цены…")
	b.syncPriceNow(ctx, chatID)
}

// syncRun — синк одного чата: скачать CSV → мерж в прайс → пометить результат.
func (b *Bot) syncRun(ctx context.Context, chatID int64) (int, error) {
	src, err := b.st.PriceSource(ctx, chatID)
	if err != nil {
		return 0, fmt.Errorf("источник не настроен — пришли ссылку: 💵 Прайс → 🌐 Google Таблица")
	}
	res := gsheet.Sync(ctx, src,
		func(items []domain.CatalogItem) (int, error) {
			return b.st.BulkUpsertCatalog(ctx, chatID, items)
		},
		func(n int, syncErr string) error {
			return b.st.MarkPriceSynced(ctx, chatID, n, syncErr)
		},
	)
	return res.Count, res.Err
}

// syncPriceNow — «🔄 Синк» с отчётом в чат.
func (b *Bot) syncPriceNow(ctx context.Context, chatID int64) {
	src, err := b.st.PriceSource(ctx, chatID)
	if err != nil {
		b.beginPriceSheet(ctx, chatID)
		return
	}
	_ = src
	_ = b.tg.SendChatAction(ctx, chatID)
	n, err := b.syncRun(ctx, chatID)
	if err != nil {
		b.textKB(ctx, chatID, "⚠ Синк не прошёл: "+err.Error(), priceSheetMenu(false))
		return
	}
	total, _ := b.st.CatalogCount(ctx, chatID)
	b.textKB(ctx, chatID, fmt.Sprintf("🔄 Синк готов: записано позиций %d (всего в прайсе %d).\nТаблица → бот синхронизированы.", n, total), priceSheetMenu(true))
}

// StartPriceSyncLoop — фоновая синхронизация прайса (v0.6): раз в interval
// тянем публичный CSV каждой подключённой таблицы. Ошибки пишутся в
// price_sources.last_error (мастер увидит их по кнопке 🌐 Google Таблица).
func (b *Bot) StartPriceSyncLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				srcs, err := b.st.ListPriceSources(ctx)
				if err != nil {
					log.Printf("синк: список источников: %v", err)
					continue
				}
				for _, src := range srcs {
					chatID := src.ChatID
					if _, err := b.syncRun(ctx, chatID); err != nil {
						log.Printf("синк чат %d: %v", chatID, err)
					}
				}
			}
		}
	}()
}

// priceSheetMenu — кнопки раздела Google Таблицы.
func priceSheetMenu(connected bool) tg.InlineKeyboardMarkup {
	rows := tg.KB{}
	if connected {
		rows = append(rows, []tg.KBButton{{Text: "🔄 Синк сейчас", CallbackData: "psync"}})
		rows = append(rows, []tg.KBButton{{Text: "🔗 Заменить ссылку", CallbackData: "psheet"}})
	} else {
		rows = append(rows, []tg.KBButton{{Text: "🌐 Подключить Google Таблицу", CallbackData: "psheet"}})
	}
	rows = append(rows, []tg.KBButton{{Text: BtnCancel, CallbackData: "cancel"}})
	return tg.Inline(rows)
}

// priceExportCSV — обратное направление (бот → мастер): прайс файлом,
// чтобы вставить в Google Таблицу вручную. Бесплатный путь вместо OAuth-push.
func (b *Bot) priceExportCSV(ctx context.Context, chatID int64) {
	items, err := b.st.ListCatalog(ctx, chatID)
	if err != nil || len(items) == 0 {
		b.text(ctx, chatID, "Прайс пуст — экспортировать нечего 🙂")
		return
	}
	var sb strings.Builder
	sb.WriteString("название;единица;цена\n")
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("%s;%s;%s\n",
			strings.ReplaceAll(it.Name, ";", ","),
			strings.ReplaceAll(it.Unit, ";", ","),
			strconv.FormatFloat(it.Price, 'f', -1, 64)))
	}
	path := filepath.Join(b.files, "exports", fmt.Sprintf("price_%d.csv", chatID))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.text(ctx, chatID, "Не собрал файл 😕")
		return
	}
	if err := os.WriteFile(path, []byte("\uFEFF"+sb.String()), 0o644); err != nil {
		log.Printf("прайс: экспорт чат %d: %v", chatID, err)
		b.text(ctx, chatID, "Не собрал файл 😕 Попробуй ещё раз.")
		return
	}
	if err := b.tg.SendDocumentFile(ctx, chatID, path, "Прайс CSV — вставь в Google Таблицу (Файл → Импорт)"); err != nil {
		b.text(ctx, chatID, "Файл не отправился 😕")
	}
}
