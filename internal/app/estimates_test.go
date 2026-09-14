// estimates_test.go — чистые функции смет в чате (v0.6): «все»/номера строк,
// статус-тексты, скрытая подготовка по словарю.
package app

import "testing"

func TestIsAllText(t *testing.T) {
	for _, s := range []string{"все", "всё", "весь", "все строки", "all", " ВСЕ "} {
		if !isAllText(s) {
			t.Fatalf("isAllText(%q) = false", s)
		}
	}
	for _, s := range []string{"1,3,5", "никогда", "все кроме", ""} {
		if isAllText(s) {
			t.Fatalf("isAllText(%q) = true", s)
		}
	}
}

func TestEstStatusText(t *testing.T) {
	cases := map[string]string{
		"sent":     "отправлена",
		"approved": "согласована",
		"done":     "закрыта",
		"draft":    "черновик",
		"unknown":  "черновик",
	}
	for in, want := range cases {
		if got := estStatusText(in); got != want {
			t.Fatalf("estStatusText(%q) = %q, ожидали %q", in, got, want)
		}
	}
}

func TestGuessHiddenName(t *testing.T) {
	if !guessHiddenName("Грунтовка стен перед шпаклёвкой") {
		t.Fatal("грунтовка обязана быть скрытой")
	}
	if guessHiddenName("Покраска стен") {
		t.Fatal("покраска не может быть скрытой")
	}
	if !guessHiddenName("гидроизоляция пола") {
		t.Fatal("гидроизоляция обязана быть скрытой")
	}
}
