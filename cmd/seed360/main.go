// Сид пилота 360: калиброванные демо-данные под chat мастера.
//
// Запуск на staging: SEED_CHAT_ID=<tg мастера> [SEED_CLIENT_ID=<tg заказчика>] ./seed360
// Повтор безопасен: сид идемпотентен, дубли не создаёт.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"proakt/internal/config"
	"proakt/internal/store"
)

func main() {
	cleanup := flag.Bool("cleanup", false, "удалить стартовые примеры и выйти")
	flag.Parse()
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}
	master, err := strconv.ParseInt(strings.TrimSpace(os.Getenv("SEED_CHAT_ID")), 10, 64)
	if err != nil || master <= 0 {
		log.Fatalf("SEED_CHAT_ID не задан или не число")
	}
	st, err := store.New(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("подключение к БД: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("миграции: %v", err)
	}
	if *cleanup {
		rep, err := st.DeleteStarterData(ctx, master)
		if err != nil {
			log.Fatalf("DeleteStarterData: %v", err)
		}
		fmt.Printf("очистка: объектов %d, цен %d\n", rep.Objects, rep.Prices)
		return
	}

	rep, err := st.Seed360(ctx, master)
	if err != nil {
		log.Fatalf("Seed360: %v", err)
	}
	fmt.Printf("объект %d: актов %d на %.2f\n", rep.ObjectID, rep.Acts, rep.Total)

	estID, estTotal, err := st.Seed360Estimate(ctx, master, rep.ObjectID)
	if err != nil {
		log.Fatalf("Seed360Estimate: %v", err)
	}
	fmt.Printf("смета %d: итог %.2f\n", estID, estTotal)

	done, err := st.Seed360EstimateProgress(ctx, master, rep.ObjectID)
	if err != nil {
		log.Fatalf("Seed360EstimateProgress: %v", err)
	}
	fmt.Printf("строк сметы помечено готовыми: %d\n", done)

	upgraded, err := st.Seed360Upgrade(ctx, master)
	if err != nil {
		log.Fatalf("Seed360Upgrade: %v", err)
	}
	fmt.Printf("актов обновлено настоящими строками: %d\n", upgraded)

	demoPayed, err := st.Seed360DemoPayments(ctx, master)
	if err != nil {
		log.Fatalf("Seed360DemoPayments: %v", err)
	}
	fmt.Printf("демо-оплат добавлено: %d\n", demoPayed)

	for _, raw := range strings.Split(os.Getenv("SEED_CLIENT_ID"), ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		tg, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || tg <= 0 {
			log.Fatalf("SEED_CLIENT_ID: %q не число", raw)
		}
		objs, err := st.ListObjects(ctx, master)
		if err != nil {
			log.Fatalf("ListObjects: %v", err)
		}
		for _, o := range objs {
			if err := st.GrantClient(ctx, o.ID, tg); err != nil {
				log.Fatalf("GrantClient %d на %d: %v", tg, o.ID, err)
			}
			fmt.Printf("заказчик %d привязан к объекту %d (%s)\n", tg, o.ID, o.Name)
		}
	}
}
