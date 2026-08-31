package ai

import (
	"encoding/json"
	"testing"
)

func TestParsePositionsJSON(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantN   int
		wantSum float64
		wantErr bool
	}{
		{"чистый массив", `[{"name":"штукатурка","qty":45,"unit":"м2","price":260}]`, 1, 11700, false},
		{"сумма без qty", `[{"name":"демонтаж","sum":2000}]`, 1, 2000, false},
		{"в кодовом блоке", "Вот позиции:\n```json\n[{\"name\":\"шпаклёвка\",\"qty\":30,\"price\":90}]\n```\nконец", 1, 2700, false},
		{"лишний текст вокруг", "Позиции: [{\"name\":\"грунтовка\",\"sum\":1500}] всё.", 1, 1500, false},
		{"пустой массив", "[]", 0, 0, false},
		{"без имени пропускаем", `[{"qty":5,"price":10},{"name":"окраска","qty":2,"price":100}]`, 1, 200, false},
		{"мусор", "модель не поняла", 0, 0, true},
		{"пусто", "", 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lines, err := ParsePositionsJSON(c.in)
			if c.wantErr {
				if err == nil {
					t.Fatalf("ожидалась ошибка, получили %v", lines)
				}
				return
			}
			if err != nil {
				t.Fatalf("не ждали ошибку: %v", err)
			}
			if len(lines) != c.wantN {
				t.Fatalf("позиций = %d, хочу %d", len(lines), c.wantN)
			}
			var sum float64
			for _, l := range lines {
				sum += l.Sum
			}
			if sum != c.wantSum {
				t.Fatalf("сумма = %v, хочу %v", sum, c.wantSum)
			}
		})
	}
}

func TestParsePositionsJSONSumOnlyFills(t *testing.T) {
	lines, err := ParsePositionsJSON(`[{"name":"демонтаж стен","sum":2000}]`)
	if err != nil || len(lines) != 1 {
		t.Fatalf("err=%v lines=%v", err, lines)
	}
	l := lines[0]
	if l.Qty != 1 || l.Price != 2000 || l.Sum != 2000 {
		t.Fatalf("sum-only не дополнился: %+v", l)
	}
}

func TestParsePositionsJSONQtyPriceComputesSum(t *testing.T) {
	lines, err := ParsePositionsJSON(`[{"name":"штукатурка","qty":45,"price":260}]`)
	if err != nil || len(lines) != 1 {
		t.Fatalf("err=%v lines=%v", err, lines)
	}
	if lines[0].Sum != 11700 {
		t.Fatalf("sum не вычислен: %+v", lines[0])
	}
}

func TestLooksLikeRefusal(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Я не вижу прикрепленной голосовой записи или аудиофайла.", true},
		{"Для того чтобы я мог расшифровать запись, пожалуйста, загрузите аудиофайл или предоставьте доступ к нему.", true},
		{"I don't see an attached audio file. Please upload the audio.", true}, // нет маркеров «не вижу» — но «загрузите аудио» нет; см. ниже
		{"Штукатурка сорок пять метров двести шестьдесят", false},
		{"Демонтаж стен две тысячи гривен", false},
		{"Грунтовка двадцать литров триста", false},
	}
	for _, c := range cases {
		if got := looksLikeRefusal(c.in); got != c.want {
			t.Errorf("looksLikeRefusal(%q) = %v, хочу %v", c.in, got, c.want)
		}
	}
}

func TestContentText(t *testing.T) {
	s := ContentText(json.RawMessage(`"привет"`))
	if s != "привет" {
		t.Fatalf("строка: %q", s)
	}
	s = ContentText(json.RawMessage(`[{"type":"text","text":"часть 1"},{"type":"text","text":"часть 2"}]`))
	if s != "часть 1\nчасть 2" {
		t.Fatalf("части: %q", s)
	}
	s = ContentText(json.RawMessage(``))
	if s != "" {
		t.Fatalf("пусто: %q", s)
	}
	s = ContentText(json.RawMessage(`[{"type":"thinking","thinking":"думаю"},{"type":"text","text":"готово"}]`))
	if s != "готово" {
		t.Fatalf("thinking+text: %q", s)
	}
}
