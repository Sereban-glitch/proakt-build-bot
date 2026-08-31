// Тесты разбора файлов прайса (XLSX/CSV) — генерируем файлы в памяти.
package pricefile

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func buildXLSX(t *testing.T, rows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	sh := f.GetSheetName(0)
	for i, row := range rows {
		cells := make([]interface{}, len(row))
		for j, c := range row {
			cells[j] = c
		}
		if err := f.SetSheetRow(sh, cell(i, 0), &cells); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func cell(row, col int) string {
	return string(rune('A'+col)) + string(rune('1'+row))
}

func TestParseXLSXTypical(t *testing.T) {
	data := buildXLSX(t, [][]any{
		{"№", "Наименование", "Ед.", "Цена"},
		{1, "Штукатурка стен", "м²", 260},
		{2, "Демонтаж перегородки", "", 2000},
		{3, "Плинтус", "м.п.", 90},
	})
	items, err := Parse(data, "price.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("позиций %d, хочу 3: %+v", len(items), items)
	}
	if items[0].Name != "штукатурка стен" || items[0].Unit != "м²" || items[0].Price != 260 {
		t.Errorf("первая позиция: %+v", items[0])
	}
	if items[1].Name != "демонтаж перегородки" || items[1].Price != 2000 {
		t.Errorf("вторая позиция: %+v", items[1])
	}
}

func TestParseXLSXOnlyNamePrice(t *testing.T) {
	data := buildXLSX(t, [][]any{
		{"вид робіт", "ціна"},
		{"Грунтовка стен", "150 грн"},
		{"Шпаклёвка", "90,5"},
	})
	items, err := Parse(data, "price.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("позиций %d, хочу 2", len(items))
	}
	if items[0].Price != 150 || items[1].Price != 90.5 {
		t.Errorf("цены: %+v", items)
	}
}

func TestParseCSV(t *testing.T) {
	csv := "Наименование;Ед;Цена\nШтукатурка;м2;260\nДемонтаж;;2000\n"
	items, err := Parse([]byte(csv), "price.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("позиций %d, хочу 2", len(items))
	}
	if items[0].Unit != "м²" || items[0].Price != 260 {
		t.Errorf("первая: %+v", items[0])
	}
	if items[1].Name != "демонтаж" {
		t.Errorf("вторая: %+v", items[1])
	}
}

func TestParseCSVComma(t *testing.T) {
	items, err := Parse([]byte("Штукатурка,260\nПлинтус,90\n"), "p.csv")
	if err != nil || len(items) != 2 {
		t.Fatalf("err=%v items=%v", err, items)
	}
}

func TestParseBOM(t *testing.T) {
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Штукатурка;260\n")...)
	items, err := Parse(data, "p.csv")
	if err != nil || len(items) != 1 || items[0].Name != "штукатурка" {
		t.Fatalf("BOM: err=%v items=%v", err, items)
	}
}

func TestParseUnknownFormat(t *testing.T) {
	if _, err := Parse([]byte("x"), "file.pdf"); err == nil {
		t.Fatal("ожидалась ошибка формата")
	}
}

func TestParseGarbageXLSX(t *testing.T) {
	if _, err := Parse([]byte("не excel"), "file.xlsx"); err == nil {
		t.Fatal("ожидалась ошибка")
	}
}

func TestParseThousandsSpace(t *testing.T) {
	csv := "Штукатурка;1 200\n"
	items, err := Parse([]byte(csv), "p.csv")
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Price != 1200 {
		t.Fatalf("цена с пробелом-тысячей: %+v", items[0])
	}
}
