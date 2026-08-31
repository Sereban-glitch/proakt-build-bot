// pricecheck — проверка файла прайса ДО отправки боту:
// pricecheck -file прайс.xlsx — печатает, что извлёк бы бот при импорте.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"proakt/internal/pricefile"
)

func main() {
	path := flag.String("file", "", "путь к файлу прайса (.xlsx/.csv)")
	flag.Parse()
	if *path == "" {
		log.Fatal("укажи -file прайс.xlsx")
	}
	data, err := os.ReadFile(*path)
	if err != nil {
		log.Fatal(err)
	}
	items, err := pricefile.Parse(data, *path)
	if err != nil {
		log.Fatalf("разбор: %v", err)
	}
	fmt.Printf("позиций: %d\n", len(items))
	for i, it := range items {
		unit := it.Unit
		if unit == "" {
			unit = "—"
		}
		fmt.Printf("%3d. %-70s %-6s %v\n", i+1, it.Name, unit, it.Price)
	}
}
