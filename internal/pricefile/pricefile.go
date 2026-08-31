// Package pricefile — разбор файла прайса (XLSX/CSV) в позиции каталога (v0.3).
// Мастер ведёт цены в Excel/Google Таблице, выгружает и присылает файлом боту.
package pricefile

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"proakt/internal/domain"
	"proakt/internal/parse"
)

// MaxRows — защита от гигантских файлов.
const MaxRows = 1000

// Parse разбирает файл по расширению имени.
func Parse(data []byte, filename string) ([]domain.CatalogItem, error) {
	name := strings.ToLower(strings.TrimSpace(filename))
	switch {
	case strings.HasSuffix(name, ".xlsx"), strings.HasSuffix(name, ".xlsm"):
		return parseXLSX(data)
	case strings.HasSuffix(name, ".csv"), strings.HasSuffix(name, ".txt"), strings.HasSuffix(name, ".tsv"):
		return parseCSV(data)
	default:
		return nil, fmt.Errorf("не знаю формат «%s» — пришли Excel (.xlsx) или CSV", filename)
	}
}

// parseXLSX — все листы, все строки: первый текст = наименование,
// крайнее правое число = цена, короткая клетка рядом = единица.
func parseXLSX(data []byte) ([]domain.CatalogItem, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("не открылся как Excel: %w", err)
	}
	defer f.Close()

	var items []domain.CatalogItem
	sheets := f.GetSheetList()
	for _, sh := range sheets {
		rows, err := f.GetRows(sh)
		if err != nil {
			continue
		}
		for _, row := range rows {
			if it, ok := rowToItem(row); ok {
				items = append(items, it)
				if len(items) >= MaxRows {
					return items, nil
				}
			}
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("в файле не нашёл ни одной строки «наименование + цена»")
	}
	return items, nil
}

// parseCSV — разделитель ; , или табуляция (по максимуму в первых строках).
func parseCSV(data []byte) ([]domain.CatalogItem, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}) // BOM
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = detectSep(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var items []domain.CatalogItem
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV: %w", err)
		}
		if it, ok := rowToItem(row); ok {
			items = append(items, it)
			if len(items) >= MaxRows {
				return items, nil
			}
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("в файле не нашёл ни одной строки «наименование + цена»")
	}
	return items, nil
}

func detectSep(data []byte) rune {
	best, bestN := ';', -1
	head := string(data[:min(len(data), 4096)])
	for _, sep := range []rune{';', ',', '\t'} {
		if n := strings.Count(head, string(sep)); n > bestN {
			best, bestN = sep, n
		}
	}
	return best
}

// cellNum — число из клетки («260», «260,50 грн», «1 200»).
func cellNum(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " "))
	s = strings.TrimSpace(strings.ReplaceAll(s, "грн", ""))
	s = strings.TrimSpace(strings.ReplaceAll(s, "₴", ""))
	// «1 200» → «1200»: пробелы-тысячи внутри числа
	if strings.Count(s, " ") == 1 && !strings.Contains(s, ",") && !strings.Contains(s, ".") {
		parts := strings.SplitN(s, " ", 2)
		if isDigits(parts[0]) && isDigits(parts[1]) {
			s = parts[0] + parts[1]
		}
	}
	if s == "" {
		return 0, false
	}
	s = strings.Replace(s, ",", ".", 1) // «260,50» → «260.50»
	if strings.Count(s, ".") > 1 {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// headerWords — маркеры шапки таблицы (строку-шапку пропускаем).
var headerWords = []string{
	"наименование", "найменування", "назва", "название", "вид робіт", "вид работ",
	"послуга", "услуга", "п/п", "№", "цена", "ціна", "од", "ед", "примечание",
}

// rowToItem — строка таблицы в позицию прайса.
func rowToItem(row []string) (domain.CatalogItem, bool) {
	type cell struct {
		idx int
		val string
	}
	var cells []cell
	for i, c := range row {
		if c = strings.TrimSpace(c); c != "" {
			cells = append(cells, cell{i, c})
		}
	}
	if len(cells) < 2 {
		return domain.CatalogItem{}, false
	}

	// цена — крайняя правая числовая клетка
	priceIdx := -1
	var price float64
	for i := len(cells) - 1; i >= 0; i-- {
		if v, ok := cellNum(cells[i].val); ok {
			price, priceIdx = v, i
			break
		}
	}
	if priceIdx <= 0 {
		return domain.CatalogItem{}, false
	}

	// наименование — первая нечисловая клетка слева от цены
	nameIdx := -1
	for i := 0; i < priceIdx; i++ {
		if _, isNum := cellNum(cells[i].val); !isNum {
			nameIdx = i
			break
		}
	}
	if nameIdx < 0 {
		return domain.CatalogItem{}, false
	}
	raw := strings.Trim(cells[nameIdx].val, " ,.-")
	if len([]rune(raw)) < 2 || isHeaderCell(raw) {
		return domain.CatalogItem{}, false
	}

	// единица — короткая клетка между наименованием и ценой
	unit := ""
	for i := nameIdx + 1; i < priceIdx; i++ {
		if u := parse.NormalizeUnit(cells[i].val); u != "" {
			unit = u
			break
		}
	}

	return domain.CatalogItem{Name: parse.NormName(raw), Unit: unit, Price: price}, true
}

// isHeaderCell — клетка выглядит как заголовок колонки.
func isHeaderCell(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, w := range headerWords {
		if s == w {
			return true
		}
	}
	return false
}
