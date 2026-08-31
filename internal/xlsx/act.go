// Package xlsx — генерация акта выполненных работ в формате XLSX (excelize).
// Документ на украинском (docs lang), формулы живые: сумму можно править в Excel.
package xlsx

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"proakt/internal/domain"
)

var unsafe = regexp.MustCompile(`[^\p{L}\p{N}_-]+`)

func slug(s string) string {
	s = strings.TrimSpace(s)
	s = unsafe.ReplaceAllString(s, "_")
	return strings.Trim(s, "_-")
}

func ptr(s string) *string { return &s }

// GenerateAct собирает XLSX-акт и возвращает путь к файлу.
func GenerateAct(dir string, brief domain.ActBrief, object domain.Object, lines []domain.ActLine, executor string, actDate time.Time) (string, error) {
	f := excelize.NewFile()
	sheet := "Акт"
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheet); err != nil {
		return "", err
	}

	// колонки: A №, B наименование, C кол-во, D ед., E цена, F сумма
	widths := []float64{5, 44, 10, 9, 12, 14}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, col, col, w)
	}

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 14, Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	subStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 11}})
	headStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "8EA9DB"}, {Type: "right", Style: 1, Color: "8EA9DB"},
			{Type: "top", Style: 1, Color: "8EA9DB"}, {Type: "bottom", Style: 1, Color: "8EA9DB"},
		},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"DDEBF7"}, Pattern: 1},
	})
	cellStyle, _ := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "B4C6E7"}, {Type: "right", Style: 1, Color: "B4C6E7"},
			{Type: "top", Style: 1, Color: "B4C6E7"}, {Type: "bottom", Style: 1, Color: "B4C6E7"},
		},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	numStyle, _ := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "B4C6E7"}, {Type: "right", Style: 1, Color: "B4C6E7"},
			{Type: "top", Style: 1, Color: "B4C6E7"}, {Type: "bottom", Style: 1, Color: "B4C6E7"},
		},
		CustomNumFmt: ptr("#,##0.00"),
	})
	boldRight, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	plainRight, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		CustomNumFmt: ptr("#,##0.00"),
	})

	// шапка
	title := fmt.Sprintf("АКТ виконаних робіт № %d", brief.ActNo)
	_ = f.SetCellValue(sheet, "A1", title)
	_ = f.MergeCell(sheet, "A1", "F1")
	_ = f.SetCellStyle(sheet, "A1", "F1", titleStyle)

	_ = f.SetCellValue(sheet, "A2", "від "+actDate.Format("02.01.2006")+" р.")
	_ = f.MergeCell(sheet, "A2", "F2")
	_ = f.SetCellStyle(sheet, "A2", "F2", subStyle)

	meta := [][2]string{
		{"Об'єкт:", object.Name},
		{"Замовник:", brief.Customer},
		{"Виконавець:", executor},
	}
	for i, row := range meta {
		r := i + 3
		cell, _ := excelize.CoordinatesToCellName(1, r)
		_ = f.SetCellValue(sheet, cell, row[0])
		_ = f.MergeCell(sheet, cell, "F"+fmt.Sprint(r))
		_ = f.SetCellStyle(sheet, cell, "F"+fmt.Sprint(r), subStyle)
	}

	// таблица позиций: заголовок в строке 7
	headerRow := 7
	headers := []string{"№", "Найменування робіт", "К-ть", "Од.", "Ціна, грн", "Сума, грн"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		_ = f.SetCellValue(sheet, cell, h)
	}
	{
		c1, _ := excelize.CoordinatesToCellName(1, headerRow)
		c2, _ := excelize.CoordinatesToCellName(6, headerRow)
		_ = f.SetCellStyle(sheet, c1, c2, headStyle)
	}

	// позиции (живые формулы в колонке F)
	r := headerRow + 1
	for i, l := range lines {
		row := r + i
		_ = f.SetCellValue(sheet, cellName(1, row), i+1)
		_ = f.SetCellValue(sheet, cellName(2, row), l.Name)
		_ = f.SetCellValue(sheet, cellName(3, row), l.Qty)
		_ = f.SetCellValue(sheet, cellName(4, row), l.Unit)
		_ = f.SetCellValue(sheet, cellName(5, row), l.Price)
		_ = f.SetCellFormula(sheet, cellName(6, row), fmt.Sprintf("=C%d*E%d", row, row))
		_ = f.SetCellStyle(sheet, cellName(1, row), cellName(2, row), cellStyle)
		_ = f.SetCellStyle(sheet, cellName(3, row), cellName(6, row), numStyle)
	}
	last := headerRow + len(lines)

	// итоги
	totalRow := last + 1
	_ = f.SetCellValue(sheet, cellName(2, totalRow), "Разом:")
	_ = f.MergeCell(sheet, cellName(2, totalRow), cellName(5, totalRow))
	_ = f.SetCellStyle(sheet, cellName(2, totalRow), cellName(5, totalRow), boldRight)
	if last >= headerRow+1 {
		_ = f.SetCellFormula(sheet, cellName(6, totalRow), fmt.Sprintf("=SUM(F%d:F%d)", headerRow+1, last))
	}
	_ = f.SetCellStyle(sheet, cellName(6, totalRow), cellName(6, totalRow), totalStyle)

	paidRow := totalRow + 1
	_ = f.SetCellValue(sheet, cellName(2, paidRow), "Оплачено:")
	_ = f.MergeCell(sheet, cellName(2, paidRow), cellName(5, paidRow))
	_ = f.SetCellStyle(sheet, cellName(2, paidRow), cellName(5, paidRow), plainRight)
	_ = f.SetCellValue(sheet, cellName(6, paidRow), brief.Paid)
	_ = f.SetCellStyle(sheet, cellName(6, paidRow), cellName(6, paidRow), numStyle)

	debtRow := paidRow + 1
	_ = f.SetCellValue(sheet, cellName(2, debtRow), "Заборгованість:")
	_ = f.MergeCell(sheet, cellName(2, debtRow), cellName(5, debtRow))
	_ = f.SetCellStyle(sheet, cellName(2, debtRow), cellName(5, debtRow), boldRight)
	_ = f.SetCellFormula(sheet, cellName(6, debtRow), fmt.Sprintf("=F%d-F%d", totalRow, paidRow))
	_ = f.SetCellStyle(sheet, cellName(6, debtRow), cellName(6, debtRow), totalStyle)

	// подписи
	signRow := debtRow + 2
	_ = f.SetCellValue(sheet, cellName(1, signRow), "Виконавець: ____________________")
	_ = f.SetCellValue(sheet, cellName(4, signRow), "Замовник: ____________________")

	// файл
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("act_%d_%s_%s.xlsx", brief.ActNo, slug(object.Name), actDate.Format("20060102"))
	path := filepath.Join(dir, name)
	if err := f.SaveAs(path); err != nil {
		return "", err
	}
	return path, nil
}

func cellName(col, row int) string {
	c, _ := excelize.CoordinatesToCellName(col, row)
	return c
}
