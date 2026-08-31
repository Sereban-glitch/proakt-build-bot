// asrlive — live-проверка голосового распознавания через реальный шлюз.
// Использование: asrlive -ogg файл.ogg -base http://127.0.0.1:18081/v1
// Вызывает ТОТ ЖЕ код, что и бот в production (internal/ai.Gateway.Transcribe).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"proakt/internal/ai"
)

func main() {
	oggPath := flag.String("ogg", "", "путь к OGG/Opus файлу")
	base := flag.String("base", "http://127.0.0.1:18081/v1", "базовый URL шлюза")
	model := flag.String("model", "gemini-3-flash", "модель для текста")
	modelASR := flag.String("model-asr", "gemini-2.5-flash", "модель для распознавания")
	flag.Parse()

	if *oggPath == "" {
		log.Fatal("укажи -ogg файл.ogg")
	}
	ogg, err := os.ReadFile(*oggPath)
	if err != nil {
		log.Fatalf("чтение %s: %v", *oggPath, err)
	}
	gw := ai.New(*base, "", *model, *modelASR)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()
	transcript, err := gw.Transcribe(ctx, ogg)
	if err != nil {
		log.Fatalf("ОШИБКА распознавания: %v", err)
	}
	fmt.Printf("OK за %v. Транскрипт:\n%s\n", time.Since(start).Round(time.Millisecond), transcript)

	lines, err := gw.ExtractPositions(ctx, transcript)
	if err != nil {
		log.Printf("позиции (сбой): %v", err)
		return
	}
	fmt.Printf("\nПозиций: %d\n", len(lines))
	for i, l := range lines {
		fmt.Printf("%d. %s · %v %s · %v → сумма %v\n", i+1, l.Name, l.Qty, l.Unit, l.Price, l.Sum)
	}
}
