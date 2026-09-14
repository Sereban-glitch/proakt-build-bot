// gsheet_test.go — разбор ссылок Google Таблиц (без сети).
package gsheet

import (
	"context"
	"errors"
	"testing"

	"proakt/internal/domain"
)

func TestParseSheetURL(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		fileID string
		gid    string
		err    bool
	}{
		{"обычная ссылка", "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjK1234567890/edit#gid=0", "1AbCdEfGhIjK1234567890", "0", false},
		{"без gid", "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjK1234567890/edit", "1AbCdEfGhIjK1234567890", "", false},
		{"view-ссылка", "https://docs.google.com/spreadsheets/d/1AbCdEfGhIjK1234567890/view", "1AbCdEfGhIjK1234567890", "", false},
		{"протолкнутая", "docs.google.com/spreadsheets/d/1AbCdEfGhIjK1234567890/edit#gid=123", "1AbCdEfGhIjK1234567890", "123", false},
		{"published", "https://docs.google.com/spreadsheets/d/e/2PACX-abc123456789def/pubhtml", "e/2PACX-abc123456789def", "", false},
		{"не таблица", "https://docs.google.com/document/d/1AbCdEfGhIjK/edit", "", "", true},
		{"не гугл", "https://example.com/spreadsheets/d/1AbCdEfGhIjK/edit", "", "", true},
		{"пустая", "   ", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fileID, gid, err := ParseSheetURL(c.raw)
			if c.err {
				if !errors.Is(err, ErrBadURL) {
					t.Fatalf("ожидали ErrBadURL, получили %v (fileID=%q)", err, fileID)
				}
				return
			}
			if err != nil {
				t.Fatalf("не ожидали ошибки: %v", err)
			}
			if fileID != c.fileID || gid != c.gid {
				t.Fatalf("fileID=%q gid=%q, ожидали %q/%q", fileID, gid, c.fileID, c.gid)
			}
		})
	}
}

func TestCsvURL(t *testing.T) {
	if got := csvURL("1ABCdefGHI123", "42"); got != "https://docs.google.com/spreadsheets/d/1ABCdefGHI123/export?format=csv&gid=42" {
		t.Fatalf("csvURL обычной таблицы: %s", got)
	}
	if got := csvURL("e/2PACX-x1234567890", ""); got != "https://docs.google.com/spreadsheets/d/e/2PACX-x1234567890/pub?output=csv&single=true" {
		t.Fatalf("csvURL published: %s", got)
	}
}

func TestSyncMarksError(t *testing.T) {
	// FetchItems без сети упадёт (фейковый fileID) — Sync обязан пометить ошибку,
	// а не запаниковать: «синк не прошёл, но бот жив».
	src := domain.PriceSource{ChatID: 1, FileID: "unknown-file-id-123456", GID: ""}
	var marked string
	res := Sync(context.Background(), src,
		func(items []domain.CatalogItem) (int, error) { return 0, nil },
		func(count int, syncErr string) error { marked = syncErr; return nil },
	)
	if res.Err == nil {
		t.Fatal("ожидали ошибку скачивания без сети")
	}
	if marked == "" {
		t.Fatal("ошибка обязана быть записана в price_sources.last_error")
	}
}
