// aiprobe — live-проверка извлечения позиций (текст → JSON) через реальный шлюз.
// Использование: aiprobe -base http://127.0.0.1:18081/v1 -text "штукатурка 45 квадратов"
// Вызывает ТОТ ЖЕ код, что и бот в production (internal/ai.Gateway.ExtractPositions).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"proakt/internal/ai"
)

func main() {
	base := flag.String("base", "http://127.0.0.1:18081/v1", "базовый URL шлюза")
	model := flag.String("model", "gemini-3-flash", "модель для текста")
	text := flag.String("text", "штукатурка сорок пять метров двести шестьдесят", "текст подрядчика")
	flag.Parse()

	gw := ai.New(*base, "", *model, "")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	lines, err := gw.ExtractPositions(ctx, *text)
	if err != nil {
		log.Fatalf("ОШИБКА извлечения: %v", err)
	}
	fmt.Printf("OK за %v. Позиций: %d\n", time.Since(start).Round(time.Millisecond), len(lines))
	for i, l := range lines {
		fmt.Printf("%d. %+v\n", i+1, l)
	}
}
