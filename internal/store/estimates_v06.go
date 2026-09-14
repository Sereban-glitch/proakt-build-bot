// estimates_v06.go — v0.6: акт из сметы за галочку, диалог с заказчиком,
// источник прайса (Google Таблица) и выгрузка данных для бэкапа.
//
// Акт из сметы — еженедельная рутина Виталика (1–2 часа в Excel): открыл
// смету → отметил строки, сделанные за неделю → акт с теми же ценами собрался
// сам, XLSX уехал заказчику, строки пометились «закрыто актом». Акт наследуется
// из сметы — никаких переписываний руками и рассинхрона цен.
package store

import (
	"context"
	"encoding/json"
	"time"

	"proakt/internal/domain"
)

// ActFromEstimate — собрать акт из выбранных строк сметы одной транзакцией.
// lineIDs пуст → берутся все незакрытые (done=false) строки с ненулевой суммой.
// Включённые строки помечаются done=true (прогресс сметы растёт, повторно
// в следующий акт они не попадут). Цены берутся из строк — как в смете.
func (s *Store) ActFromEstimate(ctx context.Context, chatID, estID int64, lineIDs []int64) (domain.ActBrief, int, error) {
	// смета обязана принадлежать чату — иначе не начинаем
	e, err := s.GetEstimate(ctx, chatID, estID)
	if err != nil {
		return domain.ActBrief{}, 0, ErrNotFound
	}
	lines, err := s.EstimateLines(ctx, e.ID)
	if err != nil {
		return domain.ActBrief{}, 0, err
	}

	picked := make([]domain.EstimateLine, 0, len(lines))
	if len(lineIDs) == 0 {
		for _, l := range lines {
			if !l.Done {
				picked = append(picked, l)
			}
		}
	} else {
		set := map[int64]bool{}
		for _, id := range lineIDs {
			set[id] = true
		}
		for _, l := range lines {
			// строки, уже закрытые актом, в новый акт не попадают и при
			// явном выборе — иначе та же работа уйдёт заказчику дважды
			if set[l.ID] && !l.Done {
				picked = append(picked, l)
			}
		}
	}
	// строки без объёма и цены не тянут в акт пустыми позициями
	filtered := picked[:0]
	for _, l := range picked {
		if l.Sum > 0.009 || l.Price > 0.009 {
			filtered = append(filtered, l)
		}
	}
	picked = filtered
	if len(picked) == 0 {
		return domain.ActBrief{}, 0, ErrNotFound // вызывающий объяснит «нечего закрывать»
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ActBrief{}, 0, err
	}
	defer tx.Rollback(ctx)

	var nextNo int
	if err := tx.QueryRow(ctx, "SELECT COALESCE(MAX(act_no),0)+1 FROM acts WHERE object_id=$1", e.ObjectID).Scan(&nextNo); err != nil {
		return domain.ActBrief{}, 0, err
	}
	var actID int64
	var date time.Time
	if err := tx.QueryRow(ctx,
		"INSERT INTO acts(object_id, act_no) VALUES($1,$2) RETURNING id, date",
		e.ObjectID, nextNo).Scan(&actID, &date); err != nil {
		return domain.ActBrief{}, 0, err
	}
	var objName, customer string
	if err := tx.QueryRow(ctx, "SELECT name, customer FROM objects WHERE id=$1", e.ObjectID).Scan(&objName, &customer); err != nil {
		return domain.ActBrief{}, 0, err
	}

	brief := domain.ActBrief{
		ID: actID, ActNo: nextNo, ObjectID: e.ObjectID,
		ObjectName: objName, Customer: customer, Date: date,
	}
	for i, l := range picked {
		if _, err := tx.Exec(ctx,
			"INSERT INTO act_lines(act_id, pos, name, qty, unit, price, sum) VALUES($1,$2,$3,$4,$5,$6,$7)",
			actID, i+1, l.Name, l.Qty, l.Unit, l.Price, l.Sum); err != nil {
			return brief, 0, err
		}
		brief.Total += l.Sum
		if _, err := tx.Exec(ctx, "UPDATE estimate_lines SET done=TRUE WHERE id=$1", l.ID); err != nil {
			return brief, 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return brief, 0, err
	}
	return brief, len(picked), nil
}

// --- комментарии к смете (диалог с заказчиком, v0.6) ----------------------------

// AddComment — комментарий мастера или заказчика (чат-скоуп для мастера,
// для заказчика проверяет share-токен в вызывающем коде).
func (s *Store) AddComment(ctx context.Context, estID int64, author, text string) (domain.EstimateComment, error) {
	var c domain.EstimateComment
	err := s.pool.QueryRow(ctx, `
INSERT INTO estimate_comments(est_id, author, text) VALUES($1,$2,$3)
RETURNING id, est_id, author, text, created_at`, estID, author, text).
		Scan(&c.ID, &c.EstID, &c.Author, &c.Text, &c.CreatedAt)
	return c, err
}

// ListComments — комментарии сметы, старые сверху (читается как переписка).
func (s *Store) ListComments(ctx context.Context, chatID, estID int64) ([]domain.EstimateComment, error) {
	rows, err := s.pool.Query(ctx, `
SELECT c.id, c.est_id, c.author, c.text, c.created_at
FROM estimate_comments c
JOIN estimates e ON e.id = c.est_id
WHERE c.est_id = $1 AND e.chat_id = $2
ORDER BY c.id`, estID, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EstimateComment
	for rows.Next() {
		var c domain.EstimateComment
		if err := rows.Scan(&c.ID, &c.EstID, &c.Author, &c.Text, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListCommentsByToken — комментарии для публичной страницы (доступ по токену).
func (s *Store) ListCommentsByToken(ctx context.Context, token string) ([]domain.EstimateComment, error) {
	rows, err := s.pool.Query(ctx, `
SELECT c.id, c.est_id, c.author, c.text, c.created_at
FROM estimate_comments c
JOIN estimates e ON e.id = c.est_id
WHERE e.share_token = $1 AND e.share_token <> ''
ORDER BY c.id`, token)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EstimateComment
	for rows.Next() {
		var c domain.EstimateComment
		if err := rows.Scan(&c.ID, &c.EstID, &c.Author, &c.Text, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ApproveEstimateByToken — заказчик принял смету: draft|sent → approved.
// Возвращает владельца, id и название для уведомления мастера.
func (s *Store) ApproveEstimateByToken(ctx context.Context, token string) (chatID, estID int64, title string, err error) {
	err = s.pool.QueryRow(ctx, `
UPDATE estimates SET status='approved'
WHERE share_token = $1 AND share_token <> '' AND status IN ('draft','sent')
RETURNING chat_id, id, title`, token).Scan(&chatID, &estID, &title)
	if err != nil {
		return 0, 0, "", ErrNotFound
	}
	return chatID, estID, title, nil
}

// ShareChatID — чат мастера по share-токену (уведомление «заказчик согласовал»).
func (s *Store) ShareChatID(ctx context.Context, token string) (int64, int64, string, error) {
	var chatID, estID int64
	var title string
	err := s.pool.QueryRow(ctx, `
SELECT chat_id, id, title FROM estimates
WHERE share_token = $1 AND share_token <> ''`, token).Scan(&chatID, &estID, &title)
	if err != nil {
		return 0, 0, "", ErrNotFound
	}
	return chatID, estID, title, nil
}

// --- источник прайса: Google Таблица (v0.6) --------------------------------------

// SetPriceSource — сохранить/обновить ссылку (одна на чат).
func (s *Store) SetPriceSource(ctx context.Context, chatID int64, url, fileID, gid string) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO price_sources(chat_id, url, file_id, gid) VALUES($1,$2,$3,$4)
ON CONFLICT (chat_id) DO UPDATE SET url=EXCLUDED.url, file_id=EXCLUDED.file_id,
  gid=EXCLUDED.gid, last_error=''`, chatID, url, fileID, gid)
	return err
}

// PriceSource — текущая ссылка чата.
func (s *Store) PriceSource(ctx context.Context, chatID int64) (domain.PriceSource, error) {
	var p domain.PriceSource
	err := s.pool.QueryRow(ctx, `
SELECT chat_id, url, file_id, gid, last_sync, last_count, last_error
FROM price_sources WHERE chat_id=$1`, chatID).
		Scan(&p.ChatID, &p.URL, &p.FileID, &p.GID, &p.LastSync, &p.LastCount, &p.LastError)
	if err != nil {
		return p, ErrNotFound
	}
	return p, nil
}

// ListPriceSources — все настроенные источники (для фоновой синхронизации).
func (s *Store) ListPriceSources(ctx context.Context) ([]domain.PriceSource, error) {
	rows, err := s.pool.Query(ctx, `
SELECT chat_id, url, file_id, gid, last_sync, last_count, last_error FROM price_sources`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PriceSource
	for rows.Next() {
		var p domain.PriceSource
		if err := rows.Scan(&p.ChatID, &p.URL, &p.FileID, &p.GID, &p.LastSync, &p.LastCount, &p.LastError); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MarkPriceSynced — результат последней синхронизации (успех или ошибка).
func (s *Store) MarkPriceSynced(ctx context.Context, chatID int64, count int, syncErr string) error {
	_, err := s.pool.Exec(ctx, `
UPDATE price_sources SET last_sync=now(), last_count=$2, last_error=$3 WHERE chat_id=$1`,
		chatID, count, syncErr)
	return err
}

// --- бэкап смет и актов (v0.6) ------------------------------------------------------

// ChatDump — всё, что жалко потерять: сметы со строками, шаблоны, прайс.
type ChatDump struct {
	ChatID    int64                `json:"chat_id"`
	CreatedAt time.Time            `json:"created_at"`
	Objects   []domain.ObjectBrief `json:"objects"`
	Estimates []estimateDump       `json:"estimates"`
	Templates []domain.Template    `json:"templates"`
	Price     []domain.CatalogItem `json:"price"`
	Acts      []actDump            `json:"acts"`
}

type estimateDump struct {
	domain.Estimate
	Lines []domain.EstimateLine `json:"lines"`
}

type actDump struct {
	domain.ActBrief
	Lines []domain.ActLine `json:"lines"`
}

// BackupChatDump — выгрузка чата в структуру для JSON-файла.
func (s *Store) BackupChatDump(ctx context.Context, chatID int64) (*ChatDump, error) {
	dump := &ChatDump{ChatID: chatID, CreatedAt: time.Now()}
	objs, err := s.ListObjects(ctx, chatID)
	if err != nil {
		return nil, err
	}
	dump.Objects = objs

	ests, err := s.ListEstimates(ctx, chatID, 0)
	if err != nil {
		return nil, err
	}
	for _, b := range ests {
		e, err := s.GetEstimate(ctx, chatID, b.ID)
		if err != nil {
			continue
		}
		lines, err := s.EstimateLines(ctx, b.ID)
		if err != nil {
			continue
		}
		dump.Estimates = append(dump.Estimates, estimateDump{Estimate: e, Lines: lines})
	}

	if tpls, err := s.ListTemplates(ctx, chatID); err == nil {
		dump.Templates = tpls
	}
	if price, err := s.ListCatalog(ctx, chatID); err == nil {
		dump.Price = price
	}

	acts, err := s.ListActs(ctx, chatID, 200)
	if err != nil {
		return nil, err
	}
	for _, a := range acts {
		lines, err := s.ActLines(ctx, a.ID)
		if err != nil {
			continue
		}
		dump.Acts = append(dump.Acts, actDump{ActBrief: a, Lines: lines})
	}
	return dump, nil
}

// Marshal — JSON-представление (с отступами: человек может открыть файл).
func (d *ChatDump) Marshal() ([]byte, error) { return json.MarshalIndent(d, "", "  ") }

// BackupChatIDs — чаты, у которых есть что бэкапить (объекты или сметы).
func (s *Store) BackupChatIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT chat_id FROM (
  SELECT chat_id FROM objects WHERE status='active'
  UNION SELECT chat_id FROM estimates
  UNION SELECT chat_id FROM catalog_items
) t`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
