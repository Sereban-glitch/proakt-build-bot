// Package rooms — шаблоны типовых помещений (v0.6, «кнопка, которая убивает
// "не знаю, с чего начать"»).
//
// Мастер на объекте: «кухня 9 м²» → готовый набор строк с нормами:
// потолок = площадь, стены ≈ периметр × высота, грунт = той же площади,
// плинтус = периметр. Мастер подправляет объёмы под реальность одним тапом —
// а техцикл (грунт → штукатурка → шпаклёвка → шлифовка → покраска) уже стоит
// в правильном порядке со скрытой подготовкой, помеченной как скрытая.
package rooms

import (
	"strconv"
	"strings"

	"proakt/internal/domain"
)

// Preset — типовое помещение.
type Preset struct {
	Key   string     `json:"key"`
	Name  string     `json:"name"`
	Hint  string     `json:"hint"`  // подсказка в UI («9 м², высота 2.7»)
	Lines []RoomLine `json:"lines"` // объёмы считаются от S/P/h
}

// RoomLine — одна работа шаблона. Source: из чего считать объём:
//
//	"S"    — площадь пола (потолок той же площади)
//	"P"    — периметр (плинтус, откосы)
//	"P*h"  — периметр × высота (стены)
//	"S*2"  — двойной слой / два прохода
//	"1"    — за всё (двери, демонтаж)
//
// fix — фиксированное количество (не зависит от размеров).
type RoomLine struct {
	Name   string  `json:"name"`
	Unit   string  `json:"unit,omitempty"`
	Source string  `json:"source"`
	Fix    float64 `json:"fix,omitempty"`
	Hidden bool    `json:"hidden,omitempty"`
}

// Скрытая подготовка по словарю (синхронизирован с guessHidden фронта —
// единая логика «что заказчик не увидит в чистовой отделке»).
func isHidden(name string) bool {
	n := strings.ToLower(name)
	for _, k := range []string{"грунт", "шпакл", "шлиф", "армир", "штроб", "укрыв", "заделк", "стык", "гипсокартон", "сетк", "уголок", "профил", "изоляц", "оттяжк", "обеспыл"} {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

// line — конструктор строки шаблона (hidden вычисляется автоматически).
func line(name, unit, source string, fix float64) RoomLine {
	return RoomLine{Name: name, Unit: unit, Source: source, Fix: fix, Hidden: isHidden(name)}
}

// Presets — стандартный набор типовых помещений (практика отделочника).
// Нормы сознательно консервативные: мастер подправит объём под реальность,
// но старт «от потолка/стен» экономит минуты на каждой комнате.
var Presets = []Preset{
	{
		Key: "room", Name: "Комната", Hint: "потолок, стены, пол — полный цикл",
		Lines: []RoomLine{
			line("демонтаж старых покрытий", "", "1", 0),
			line("штукатурка стен", "м²", "P*h", 0),
			line("грунтовка стен", "м²", "P*h", 0),
			line("шпаклёвка стен в 2 слоя", "м²", "P*h", 0),
			line("шлифовка стен", "м²", "P*h", 0),
			line("грунтовка потолка", "м²", "S", 0),
			line("шпаклёвка потолка в 2 слоя", "м²", "S", 0),
			line("шлифовка потолка", "м²", "S", 0),
			line("покраска потолка", "м²", "S", 0),
			line("стяжка пола", "м²", "S", 0),
			line("покраска стен", "м²", "P*h", 0),
			line("плинтус", "м.п", "P", 0),
		},
	},
	{
		Key: "kitchen", Name: "Кухня", Hint: "фартук, стены под плитку/покраску",
		Lines: []RoomLine{
			line("демонтаж старых покрытий", "", "1", 0),
			line("штукатурка стен", "м²", "P*h", 0),
			line("грунтовка стен", "м²", "P*h", 0),
			line("шпаклёвка стен в 2 слоя", "м²", "P*h", 0),
			line("шлифовка стен", "м²", "P*h", 0),
			line("грунтовка потолка", "м²", "S", 0),
			line("шпаклёвка потолка в 2 слоя", "м²", "S", 0),
			line("покраска потолка", "м²", "S", 0),
			line("стяжка пола", "м²", "S", 0),
			line("плитка на пол", "м²", "S", 0),
			line("фартук (кухонный)", "м²", "1", 0),
			line("плинтус", "м.п", "P", 0),
		},
	},
	{
		Key: "bath", Name: "Ванная", Hint: "гидроизоляция, плитка, сантехника",
		Lines: []RoomLine{
			line("демонтаж старой плитки", "м²", "S", 0),
			line("штукатурка стен", "м²", "P*h", 0),
			line("гидроизоляция пола", "м²", "S", 0),
			line("гидроизоляция стен (мокрая зона)", "м²", "P*h", 0),
			line("стяжка пола", "м²", "S", 0),
			line("плитка на пол", "м²", "S", 0),
			line("плитка на стены", "м²", "P*h", 0),
			line("затирка швов", "м²", "S", 0),
			line("сантехника, установка", "", "1", 0),
		},
	},
	{
		Key: "wc", Name: "Туалет", Hint: "маленькое помещение — плитка и сантехника",
		Lines: []RoomLine{
			line("демонтаж старой плитки", "м²", "S", 0),
			line("штукатурка стен", "м²", "P*h", 0),
			line("гидроизоляция пола", "м²", "S", 0),
			line("плитка на пол", "м²", "S", 0),
			line("плитка на стены", "м²", "P*h", 0),
			line("сантехника, установка", "", "1", 0),
		},
	},
	{
		Key: "hall", Name: "Коридор / прихожая", Hint: "стены + пол, часто ламинат",
		Lines: []RoomLine{
			line("демонтаж старых покрытий", "", "1", 0),
			line("штукатурка стен", "м²", "P*h", 0),
			line("грунтовка стен", "м²", "P*h", 0),
			line("шпаклёвка стен в 2 слоя", "м²", "P*h", 0),
			line("шлифовка стен", "м²", "P*h", 0),
			line("стяжка пола", "м²", "S", 0),
			line("ламинат, укладка", "м²", "S", 0),
			line("покраска стен", "м²", "P*h", 0),
			line("плинтус", "м.п", "P", 0),
		},
	},
	{
		Key: "balcony", Name: "Балкон / лоджия", Hint: "утепление, мелкий цикл",
		Lines: []RoomLine{
			line("демонтаж старых покрытий", "", "1", 0),
			line("утепление стен", "м²", "P*h", 0),
			line("гипсокартон, монтаж", "м²", "P*h", 0),
			line("шпаклёвка стен в 2 слоя", "м²", "P*h", 0),
			line("шлифовка стен", "м²", "P*h", 0),
			line("покраска стен", "м²", "P*h", 0),
			line("стяжка пола", "м²", "S", 0),
			line("ламинат, укладка", "м²", "S", 0),
		},
	},
}

// PresetByKey — пресет по ключу (ok=false — неизвестный ключ).
func PresetByKey(key string) (Preset, bool) {
	for _, p := range Presets {
		if p.Key == key {
			return p, true
		}
	}
	return Preset{}, false
}

// Build — объёмы по нормам: S — площадь, h — высота, P — периметр
// (если периметр 0, оцениваем по площади квадратной комнаты: 4·√S).
func Build(p Preset, area, height, perimeter float64) []domain.EstimateLine {
	if perimeter <= 0 && area > 0 {
		perimeter = 4 * sqrtOf(area)
	}
	out := make([]domain.EstimateLine, 0, len(p.Lines))
	for _, rl := range p.Lines {
		var qty float64
		switch rl.Source {
		case "S":
			qty = area
		case "P":
			qty = perimeter
		case "P*h":
			qty = perimeter * height
		case "1":
			qty = 1
		default:
			qty = parseFactor(rl.Source, area, height, perimeter)
		}
		if rl.Fix > 0 {
			qty = rl.Fix
		}
		qty = round2(qty)
		out = append(out, domain.EstimateLine{
			Name: rl.Name, Qty: qty, Unit: rl.Unit, Hidden: rl.Hidden, Note: p.Name,
		})
	}
	return out
}

// parseFactor — простые выражения вида "S*2", "P*h*1.15" (запас на раскрой).
func parseFactor(expr string, s, h, p float64) float64 {
	expr = strings.TrimSpace(expr)
	mult := 1.0
	for _, part := range strings.Split(expr, "*") {
		part = strings.TrimSpace(part)
		var v float64
		switch part {
		case "S":
			v = s
		case "h":
			v = h
		case "P":
			v = p
		case "":
			continue
		default:
			f, err := strconv.ParseFloat(strings.Replace(part, ",", ".", 1), 64)
			if err != nil {
				return 0
			}
			v = f
		}
		mult *= v
	}
	return mult
}

// EstimateName — название сметы/блока от пресета: «Кухня 9 м²».
func EstimateName(key string, area float64) string {
	p, ok := PresetByKey(key)
	if !ok {
		return "Помещение"
	}
	return p.Name + " " + trimNum(area) + " м²"
}

// --- мелкие хелперы (без math в сигнатурах — легко тестировать) ----------------

func sqrtOf(x float64) float64 {
	// Герон: точности 3 итераций хватает для оценки периметра
	if x <= 0 {
		return 0
	}
	r := x
	for i := 0; i < 24; i++ {
		r = (r + x/r) / 2
	}
	return r
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	if err != nil {
		return 0
	}
	return v
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func trimNum(v float64) string {
	if v == float64(int64(v)) {
		return itoa(int64(v))
	}
	return ftoa2(v)
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func ftoa2(v float64) string {
	whole := int64(v)
	frac := int64((v - float64(whole)) * 100)
	return itoa(whole) + "," + pad2(frac)
}

func pad2(v int64) string {
	if v < 10 {
		return "0" + itoa(v)
	}
	return itoa(v)
}
