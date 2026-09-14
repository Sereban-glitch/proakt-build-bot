// Package gsheet — синхронизация прайса с Google Таблицей (v0.6).
//
// Путь без ключей и OAuth (ограничение проекта — только бесплатные сервисы):
// Google Таблица → «Файл → Поделиться → Все, у кого есть ссылка: Читатель»
// (или «Файл → Опубликовать в сети») → бот скачивает CSV-экспорт по прямой
// ссылке и мержит позиции в прайс чата. Таблица = источник истины: правки
// цен вносятся там, где удобно мастеру, и подтягиваются ботом по кнопке
// и по расписанию.
//
// TODO(v0.7): запись в таблицу из бота (push) требует OAuth2 — помечено как
// бесплатный путь через gcloud-приложение владельца; сейчас экспорт CSV в
// чат по команде «Выгрузить прайс» закрывает обратное направление руками.
package gsheet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"proakt/internal/domain"
	"proakt/internal/parse"
	"proakt/internal/pricefile"
)

var (
	// ErrBadURL — не похоже на ссылку Google Таблицы.
	ErrBadURL = errors.New("это не ссылка на Google Таблицу — жми «Поделиться» и пришли ссылку целиком")
	// ErrHTTP — таблица не отдалась (нет публичного доступа?).
	ErrHTTP = errors.New("Google не отдал таблицу — проверь доступ «все, у кого есть ссылка»")
)

var (
	reFile = regexp.MustCompile(`/spreadsheets/d/([A-Za-z0-9_-]{10,})`)
	reGID  = regexp.MustCompile(`[#&?]gid=([0-9]+)`)
)

// ParseSheetURL — вытащить fileID и gid из любой из ссылок:
//
//	https://docs.google.com/spreadsheets/d/<ID>/edit#gid=0
//	https://docs.google.com/spreadsheets/d/e/<pubID>/pubhtml
//	https://docs.google.com/spreadsheets/d/<ID>/view
//
// Для /d/e/<pubID> (опубликованная ссылка) fileID вернётся как e/<pubID> —
// CSV-экспорт для неё строится от pub?gid=…&single=true&output=csv.
func ParseSheetURL(raw string) (fileID, gid string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", ErrBadURL
	}
	// «протолкнули» ссылку без https://
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, perr := url.Parse(raw)
	if perr != nil || !strings.Contains(u.Host, "google.") {
		return "", "", ErrBadURL
	}
	if !strings.Contains(u.Path, "spreadsheets") {
		return "", "", ErrBadURL
	}
	if m := reGID.FindStringSubmatch(raw); m != nil {
		gid = m[1]
	}
	if m := reFile.FindStringSubmatch(u.Path); m != nil {
		return m[1], gid, nil
	}
	// published: /spreadsheets/d/e/<PUBID>/pubhtml — вытащим pubID
	if strings.Contains(u.Path, "/d/e/") && strings.Contains(u.Path, "pub") {
		parts := strings.Split(u.Path, "/")
		for i, p := range parts {
			if p == "e" && i+1 < len(parts) && len(parts[i+1]) >= 10 {
				return "e/" + parts[i+1], gid, nil
			}
		}
	}
	return "", "", ErrBadURL
}

// csvURL — прямая ссылка на CSV-экспорт листа (работает для публичных таблиц).
func csvURL(fileID, gid string) string {
	if strings.HasPrefix(fileID, "e/") { // «Опубликовать в сети»
		u := "https://docs.google.com/spreadsheets/d/" + fileID + "/pub?output=csv&single=true"
		if gid != "" {
			u += "&gid=" + url.QueryEscape(gid)
		}
		return u
	}
	u := "https://docs.google.com/spreadsheets/d/" + fileID + "/export?format=csv"
	if gid != "" {
		u += "&gid=" + url.QueryEscape(gid)
	}
	return u
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// FetchItems — скачать CSV и разобрать в позиции прайса.
// Разбор переиспользует pricefile.Parse (автодетект `;`/`,`/таб, BOM,
// «1 200», «260,50 грн», строки-шапки) — формат таблицы мастера может
// быть каким угодно, цена обычно крайняя правая числовая колонка.
func FetchItems(ctx context.Context, fileID, gid string) ([]domain.CatalogItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, csvURL(fileID, gid), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("скачивание: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrHTTP
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	items, err := pricefile.Parse(data, "sheet.csv")
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("в таблице не нашлось позиций (нужны колонки: название, [единица], цена)")
	}
	return items, nil
}

// SyncResult — что произошло при синхронизации.
type SyncResult struct {
	Count    int
	Duration time.Duration
	Err      error
}

// Sync — синхронизация одного чата: скачать → смержить → пометить результат.
// Merge-семантика: существующие цены обновляются, новые добавляются, ничего
// не удаляется (мастер мог вести часть позиций только в боте).
func Sync(ctx context.Context, src domain.PriceSource, upsert func(items []domain.CatalogItem) (int, error), mark func(count int, syncErr string) error) SyncResult {
	start := time.Now()
	items, err := FetchItems(ctx, src.FileID, src.GID)
	if err != nil {
		_ = mark(0, err.Error())
		return SyncResult{Duration: time.Since(start), Err: err}
	}
	n, err := upsert(items)
	if err != nil {
		_ = mark(0, err.Error())
		return SyncResult{Duration: time.Since(start), Err: err}
	}
	_ = mark(n, "")
	return SyncResult{Count: n, Duration: time.Since(start)}
}

// PriceFromItems — нормализация позиций (контракт для будущего «вставить
// прайс-лист как смету-шаблон»).
func PriceFromItems(items []domain.CatalogItem) []domain.CatalogItem {
	out := make([]domain.CatalogItem, len(items))
	for i, it := range items {
		out[i] = domain.CatalogItem{Name: parse.NormName(it.Name), Unit: parse.NormalizeUnit(it.Unit), Price: it.Price}
	}
	return out
}
