// estimate.go — экспорт сметы в XLSX ровно в том формате, в котором мастер
// ведёт свои таблицы (v0.5): вид робіт | м²,шт | м.п | ціна за одиницю |
// сума | примітки + «сума з розділів». Формулы живые:qty × цена пересчитается
// в Excel, если мастер подправит количество на месте.
package xlsx

import (
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"

	"proakt/internal/domain"
)

// unitCols — в живой таблице мастера количество стоит в одной из двух колонок:
// B «м²,шт» (площадь/штуки) или C «м.п» (погонные метры). Сохраняем это.
func unitCols(l domain.EstimateLine) (colB, colC float64) {
	switch l.Unit {
	case "м.п", "м.п.", "мп":
		return 0, l.Qty
	default:
		return l.Qty, 0
	}
}

// WriteEstimate — пишет смету в w (http.ResponseWriter или файл).
// brief — деньги с коэффициентом; lines — позиции с базовыми суммами.
func WriteEstimate(w io.Writer, brief domain.EstimateBrief, lines []domain.EstimateLine) error {
	f := excelize.NewFile()
	sheet := "Смета"
	_ = f.SetSheetName(f.GetSheetName(0), sheet)

	widths := []float64{52, 9, 9, 13, 13, 24, 13}
	for i, wd := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, col, col, wd)
	}

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 13, Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	subStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Size: 10, Italic: true}})
	headStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"DDEBF7"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "8EA9DB"}, {Type: "right", Style: 1, Color: "8EA9DB"},
			{Type: "top", Style: 1, Color: "8EA9DB"}, {Type: "bottom", Style: 1, Color: "8EA9DB"},
		},
	})
	cellStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "B4C6E7"}, {Type: "right", Style: 1, Color: "B4C6E7"},
			{Type: "top", Style: 1, Color: "B4C6E7"}, {Type: "bottom", Style: 1, Color: "B4C6E7"},
		},
	})
	cellHidden, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"FDF2EC"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "F4B183"}, {Type: "right", Style: 1, Color: "F4B183"},
			{Type: "top", Style: 1, Color: "F4B183"}, {Type: "bottom", Style: 1, Color: "F4B183"},
		},
	})
	numFmt := "#,##0.00"
	numStyle, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "B4C6E7"}, {Type: "right", Style: 1, Color: "B4C6E7"},
			{Type: "top", Style: 1, Color: "B4C6E7"}, {Type: "bottom", Style: 1, Color: "B4C6E7"},
		},
	})
	numHidden, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Fill:         excelize.Fill{Type: "pattern", Color: []string{"FDF2EC"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "F4B183"}, {Type: "right", Style: 1, Color: "F4B183"},
			{Type: "top", Style: 1, Color: "F4B183"}, {Type: "bottom", Style: 1, Color: "F4B183"},
		},
	})
	boldRight, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: &excelize.Alignment{Horizontal: "right"}})
	totalStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, CustomNumFmt: &numFmt})

	// шапка
	_ = f.SetCellValue(sheet, "A1", "Смета — "+brief.Title)
	_ = f.MergeCell(sheet, "A1", "G1")
	_ = f.SetCellStyle(sheet, "A1", "G1", titleStyle)

	sub := fmt.Sprintf("Об'єкт: %s · позицій: %d", brief.ObjectName, brief.Lines)
	if brief.Coeff != 1 {
		sub += fmt.Sprintf(" · коефіцієнт складності: %.2f", brief.Coeff)
	}
	_ = f.SetCellValue(sheet, "A2", sub)
	_ = f.MergeCell(sheet, "A2", "G2")
	_ = f.SetCellStyle(sheet, "A2", "G2", subStyle)
	if brief.Note != "" {
		_ = f.SetCellValue(sheet, "A3", brief.Note)
		_ = f.MergeCell(sheet, "A3", "G3")
		_ = f.SetCellStyle(sheet, "A3", "G3", subStyle)
	}

	// заголовок таблицы — как у мастера; скрытые работы — с тёплой шапкой
	const headerRow = 5
	headers := []string{"вид робіт", "м²,шт", "м.п", "ціна за одиницю", "сума", "примітки", "сума з розділів"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		_ = f.SetCellValue(sheet, cell, h)
	}
	_ = f.SetCellStyle(sheet, "A"+fmt.Sprint(headerRow), "G"+fmt.Sprint(headerRow), headStyle)

	// позиции: скрытые работы подсвечены тёплым фоном и помечены
	r := headerRow + 1
	visibleSumRows := []string{}
	hiddenSumRows := []string{}
	for _, l := range lines {
		row := r
		b, c := unitCols(l)
		base := cellStyle
		num := numStyle
		prefix := ""
		if l.Hidden {
			base = cellHidden
			num = numHidden
			prefix = " ▪ "
		}
		_ = f.SetCellValue(sheet, cellName(1, row), prefix+l.Name)
		_ = f.SetCellValue(sheet, cellName(2, row), b)
		_ = f.SetCellValue(sheet, cellName(3, row), c)
		_ = f.SetCellValue(sheet, cellName(4, row), l.Price)
		_ = f.SetCellValue(sheet, cellName(6, row), l.Note)
		if brief.Coeff != 1 {
			_ = f.SetCellFormula(sheet, cellName(5, row), fmt.Sprintf("=ROUND((B%d+C%d)*D%d*%.2f,2)", row, row, row, brief.Coeff))
		} else {
			_ = f.SetCellFormula(sheet, cellName(5, row), fmt.Sprintf("=(B%d+C%d)*D%d", row, row, row))
		}
		_ = f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), base)
		_ = f.SetCellStyle(sheet, cellName(6, row), cellName(6, row), base)
		_ = f.SetCellStyle(sheet, cellName(2, row), cellName(5, row), num)
		ref := fmt.Sprintf("E%d", row)
		if l.Hidden {
			hiddenSumRows = append(hiddenSumRows, ref)
		} else {
			visibleSumRows = append(visibleSumRows, ref)
		}
		r++
	}

	sumEq := func(rows []string) string {
		if len(rows) == 0 {
			return ""
		}
		return "=" + strings.Join(rows, "+")
	}

	// итоги: по разделам «видимых» и «скрытых» — как сума з розділів, только честнее
	totalRow := r
	_ = f.SetCellValue(sheet, cellName(1, totalRow), "РАЗОМ за сметою:")
	_ = f.MergeCell(sheet, cellName(1, totalRow), cellName(4, totalRow))
	_ = f.SetCellStyle(sheet, cellName(1, totalRow), cellName(4, totalRow), boldRight)
	if eq := sumEq(append(append([]string{}, visibleSumRows...), hiddenSumRows...)); eq != "" {
		_ = f.SetCellFormula(sheet, cellName(5, totalRow), eq)
	}
	_ = f.SetCellStyle(sheet, cellName(5, totalRow), cellName(5, totalRow), totalStyle)

	row := totalRow
	if len(visibleSumRows) > 0 {
		row++
		_ = f.SetCellValue(sheet, cellName(1, row), "чистові роботи (видно замовнику):")
		_ = f.MergeCell(sheet, cellName(1, row), cellName(4, row))
		_ = f.SetCellStyle(sheet, cellName(1, row), cellName(4, row), boldRight)
		_ = f.SetCellFormula(sheet, cellName(5, row), sumEq(visibleSumRows))
		_ = f.SetCellStyle(sheet, cellName(5, row), cellName(5, row), totalStyle)
	}
	if len(hiddenSumRows) > 0 {
		row++
		_ = f.SetCellValue(sheet, cellName(1, row), "скриті/підготовчі роботи (підготовка під чистову):")
		_ = f.MergeCell(sheet, cellName(1, row), cellName(4, row))
		_ = f.SetCellStyle(sheet, cellName(1, row), cellName(4, row), boldRight)
		_ = f.SetCellFormula(sheet, cellName(5, row), sumEq(hiddenSumRows))
		_ = f.SetCellStyle(sheet, cellName(5, row), cellName(5, row), totalStyle)
	}

	// подпись
	row += 2
	_ = f.SetCellValue(sheet, cellName(1, row), "▪ — скриті роботи: підготовка, яку замовник не бачить у фініші, але вона є в кошторисі")
	_ = f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), subStyle)

	_, err := f.WriteTo(w)
	return err
}

// EstimateFileName — человекочитаемое имя файла.
func EstimateFileName(brief domain.EstimateBrief) string {
	return fmt.Sprintf("smeta_%d_%s.xlsx", brief.ID, slug(brief.Title))
}
