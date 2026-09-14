// rooms_test.go — нормы типовых помещений: потолок = площадь,
// стены = периметр × высота, плинтус = периметр, скрытая подготовка.
package rooms

import (
	"strings"
	"testing"
)

func TestBuildKitchen(t *testing.T) {
	p, ok := PresetByKey("kitchen")
	if !ok {
		t.Fatal("пресет кухни не найден")
	}
	// кухня 9 м², высота 2.7, периметр 12
	lines := Build(p, 9, 2.7, 12)
	if len(lines) == 0 {
		t.Fatal("Build вернул пустой набор")
	}
	byName := map[string]float64{}
	for _, l := range lines {
		byName[l.Name] = l.Qty
	}
	if byName["стяжка пола"] != 9 { // потолок/пол = площадь
		t.Fatalf("площадь пола должна перейти в стяжку: %.2f", byName["стяжка пола"])
	}
	if byName["штукатурка стен"] != 32.4 { // 12 × 2.7
		t.Fatalf("стены должны быть P×h = 32.4, получили %.2f", byName["штукатурка стен"])
	}
	if byName["плинтус"] != 12 { // периметр
		t.Fatalf("плинтус = периметр (12), получили %.2f", byName["плинтус"])
	}
}

func TestBuildPerimeterAuto(t *testing.T) {
	p, _ := PresetByKey("room")
	// периметр 0 → ≈4·√S (квадратная комната): 25 м² → 20 м
	lines := Build(p, 25, 2.7, 0)
	var walls float64
	for _, l := range lines {
		if l.Name == "штукатурка стен" {
			walls = l.Qty
		}
	}
	if walls < 53 || walls > 55 { // 20 × 2.7 = 54
		t.Fatalf("авто-периметр: ожидали ≈54 м² стен, получили %.2f", walls)
	}
}

func TestHiddenMarks(t *testing.T) {
	p, _ := PresetByKey("room")
	lines := Build(p, 20, 2.7, 18)
	for _, l := range lines {
		if strings.Contains(l.Name, "грунтовка") && !l.Hidden {
			t.Fatalf("грунтовка должна быть скрытой подготовкой: %s", l.Name)
		}
		if strings.Contains(l.Name, "покраска") && l.Hidden {
			t.Fatalf("покраска не может быть скрытой: %s", l.Name)
		}
	}
}

func TestAllPresetsBuildable(t *testing.T) {
	if len(Presets) < 5 {
		t.Fatalf("ожидал минимум 5 пресетов, получили %d", len(Presets))
	}
	for _, p := range Presets {
		lines := Build(p, 10, 2.7, 14)
		if len(lines) == 0 {
			t.Fatalf("пресет %s не строится", p.Key)
		}
		for _, l := range lines {
			if l.Name == "" {
				t.Fatalf("пустое имя работы в %s", p.Key)
			}
		}
	}
}

func TestEstimateName(t *testing.T) {
	got := EstimateName("kitchen", 9)
	if got != "Кухня 9 м²" {
		t.Fatalf("EstimateName: %s", got)
	}
}

func TestParseFactorStock(t *testing.T) {
	if v := parseFactor("S*2", 10, 2.7, 12); v != 20 {
		t.Fatalf("S*2: %.2f", v)
	}
	if v := parseFactor("P*h*1.15", 10, 2.7, 12); v < 37.2 || v > 37.3 {
		t.Fatalf("P*h*1.15: %.3f", v)
	}
}
