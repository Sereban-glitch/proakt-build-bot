// Package domain — сущности предметной области (акт, объект, оплата, фото).
package domain

import "time"

// Object — объект (квартира/дом), рабочее пространство привязано к chat_id.
type Object struct {
	ID        int64     `json:"id"`
	ChatID    int64     `json:"chat_id"`
	Name      string    `json:"name"`
	Customer  string    `json:"customer"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ActLine — позиция акта.
type ActLine struct {
	Pos   int     `json:"pos"`
	Name  string  `json:"name"`
	Qty   float64 `json:"qty"`
	Unit  string  `json:"unit"`
	Price float64 `json:"price"`
	Sum   float64 `json:"sum"`
}

// Act — акт выполненных работ.
type Act struct {
	ID       int64     `json:"id"`
	ObjectID int64     `json:"object_id"`
	ActNo    int       `json:"act_no"`
	Status   string    `json:"status"`
	Date     time.Time `json:"date"`
}

// ActBrief — сводка акта с деньгами (для списков и долгов).
type ActBrief struct {
	ID         int64     `json:"id"`
	ActNo      int       `json:"act_no"`
	ObjectID   int64     `json:"object_id"`
	ObjectName string    `json:"object_name"`
	Customer   string    `json:"customer"`
	Total      float64   `json:"total"`
	Paid       float64   `json:"paid"`
	Photos     int       `json:"photos"` // фото, привязанные к акту (v0.3.7)
	Date       time.Time `json:"date"`
}

// Balance — сколько осталось оплатить.
func (a ActBrief) Balance() float64 { return a.Total - a.Paid }

// Payment — оплата по акту.
type Payment struct {
	ID     int64     `json:"id"`
	ActID  int64     `json:"act_id"`
	Amount float64   `json:"amount"`
	Note   string    `json:"note"`
	At     time.Time `json:"at"`
}

// PaymentRec — оплата с контекстом акта/объекта (для удаления ошибочной
// оплаты: «🗑 10 000 грн от 05.08, акт №3 · ЖК Сонячний», v0.3.8).
type PaymentRec struct {
	ID      int64     `json:"id"`
	ActID   int64     `json:"act_id"`
	Amount  float64   `json:"amount"`
	At      time.Time `json:"at"`
	ActNo   int       `json:"act_no"`
	ObjName string    `json:"obj_name"`
}

// PhotoRec — фото скрытых работ (привязка к акту или объекту).
type PhotoRec struct {
	ID        int64     `json:"id"`
	ActID     *int64    `json:"act_id"`
	ObjectID  int64     `json:"object_id"`
	FileID    string    `json:"file_id"`
	FilePath  string    `json:"file_path"`
	Caption   string    `json:"caption"`
	ActNo     int       `json:"act_no"`     // 0 — фото без акта (v0.3.7)
	CreatedAt time.Time `json:"created_at"` // когда снято/прислано (v0.3.7)
}

// ObjectBrief — объект с итогами по актам (для списка «объекты с деньгами», v0.3.5):
// строитель видит предварительный итог по каждому объекту, не открывая акты.
type ObjectBrief struct {
	Object
	Acts   int     `json:"acts"`
	Total  float64 `json:"total"`
	Paid   float64 `json:"paid"`
	Photos int     `json:"photos"` // фото объекта, для кнопки «📷 показать» (v0.3.7)
}

// Debt — сколько осталось оплатить по объекту.
func (o ObjectBrief) Debt() float64 { return o.Total - o.Paid }

// CatalogItem — позиция прайс-листа (цены мастера, v0.3).
// Name хранится нормализованным (нижний регистр, ё→е, без двойных пробелов) —
// чтобы «Штукатурка» и «штукатурка» были одной позицией.
type CatalogItem struct {
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Price float64 `json:"price"`
}

// DraftLine — позиция в черновике (FSM), сериализуется в JSON.
type DraftLine struct {
	Name  string  `json:"name"`
	Qty   float64 `json:"qty"`
	Unit  string  `json:"unit,omitempty"`
	Price float64 `json:"price"`
	Sum   float64 `json:"sum"`
}

// Estimate — смета объекта (v0.5): план работ до начала ремонта.
// Формат повторяет живую таблицу мастера: вид робіт | м²,шт | м.п | ціна |
// сума | примітки. Coeff — коэффициент сложности (потолки выше 270 см,
// лестницы, сжатые сроки — практика прайс-листа мастера).
type Estimate struct {
	ID         int64     `json:"id"`
	ObjectID   int64     `json:"object_id"`
	ChatID     int64     `json:"chat_id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"` // draft | sent | approved | done
	Coeff      float64   `json:"coeff"`
	Note       string    `json:"note"`
	ShareToken string    `json:"share_token"` // токен публичной ссылки для заказчика ("" — не создана)
	CreatedAt  time.Time `json:"created_at"`
}

// EstimateLine — позиция сметы.
// Hidden — «скрытая/подготовительная» работа (грунтовка, армировка, штробы):
// заказчик её не видит в чистовом ремонте, но именно она объясняет,
// «на что уходят деньги». Done — позиция закрыта актом (прогресс сметы).
type EstimateLine struct {
	ID     int64   `json:"id"`
	EstID  int64   `json:"est_id"`
	Pos    int     `json:"pos"`
	Name   string  `json:"name"`
	Qty    float64 `json:"qty"`
	Unit   string  `json:"unit"` // м² | м.п | шт | "" (за всё)
	Price  float64 `json:"price"`
	Sum    float64 `json:"sum"` // qty × price — базовая, коэффициент применяется в итогах
	Hidden bool    `json:"hidden"`
	Done   bool    `json:"done"`
	Note   string  `json:"note"`
}

// EstimateBrief — смета с деньгами и прогрессом (для списков и шапки).
type EstimateBrief struct {
	ID          int64     `json:"id"`
	ObjectID    int64     `json:"object_id"`
	ObjectName  string    `json:"object_name"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	Coeff       float64   `json:"coeff"`
	Note        string    `json:"note"`
	ShareToken  string    `json:"share_token"`
	Lines       int       `json:"lines"`
	Total       float64   `json:"total"`        // вся смета (с коэффициентом)
	Visible     float64   `json:"visible"`      // чистовые работы (то, что видит заказчик)
	Hidden      float64   `json:"hidden"`       // скрытая подготовка
	DoneSum     float64   `json:"done_sum"`     // закрыто актами
	HiddenShare int       `json:"hidden_share"` // % скрытых работ от общего — «почему так дорого»
	CreatedAt   time.Time `json:"created_at"`
}

// Progress — доля сметы, закрытая актами (0..1).
func (e EstimateBrief) Progress() float64 {
	if e.Total < 0.01 {
		return 0
	}
	p := e.DoneSum / e.Total
	if p > 1 {
		return 1
	}
	return p
}

// Template — шаблон набора работ (v0.5, «админка»): «Покраска комнаты
// под ключ» = упорядоченный список позиций из прайса. Один тап — и весь
// технологический цикл (грунт → шпаклёвка → стеклохолст → покраска)
// в смете, без ручного перебивания строк из Excel.
type Template struct {
	ID    int64          `json:"id"`
	Name  string         `json:"name"`
	Lines []TemplateLine `json:"lines"`
}

// TemplateLine — позиция шаблона (ссылается на прайс по имени).
type TemplateLine struct {
	Name   string  `json:"name"`
	Qty    float64 `json:"qty"`
	Unit   string  `json:"unit,omitempty"`
	Price  float64 `json:"price"`
	Hidden bool    `json:"hidden"`
}
