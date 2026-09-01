// Package domain — сущности предметной области (акт, объект, оплата, фото).
package domain

import "time"

// Object — объект (квартира/дом), рабочее пространство привязано к chat_id.
type Object struct {
	ID        int64
	ChatID    int64
	Name      string
	Customer  string
	Status    string
	CreatedAt time.Time
}

// ActLine — позиция акта.
type ActLine struct {
	Pos   int
	Name  string
	Qty   float64
	Unit  string
	Price float64
	Sum   float64
}

// Act — акт выполненных работ.
type Act struct {
	ID       int64
	ObjectID int64
	ActNo    int
	Status   string
	Date     time.Time
}

// ActBrief — сводка акта с деньгами (для списков и долгов).
type ActBrief struct {
	ID         int64
	ActNo      int
	ObjectID   int64
	ObjectName string
	Customer   string
	Total      float64
	Paid       float64
	Photos     int // фото, привязанные к акту (v0.3.7)
}

// Balance — сколько осталось оплатить.
func (a ActBrief) Balance() float64 { return a.Total - a.Paid }

// Payment — оплата по акту.
type Payment struct {
	ID     int64
	ActID  int64
	Amount float64
	Note   string
	At     time.Time
}

// PaymentRec — оплата с контекстом акта/объекта (для удаления ошибочной
// оплаты: «🗑 10 000 грн от 05.08, акт №3 · ЖК Сонячний», v0.3.8).
type PaymentRec struct {
	ID      int64
	ActID   int64
	Amount  float64
	At      time.Time
	ActNo   int
	ObjName string
}

// PhotoRec — фото скрытых работ (привязка к акту или объекту).
type PhotoRec struct {
	ID        int64
	ActID     *int64
	ObjectID  int64
	FileID    string
	FilePath  string
	Caption   string
	ActNo     int       // 0 — фото без акта (v0.3.7)
	CreatedAt time.Time // когда снято/прислано (v0.3.7)
}

// ObjectBrief — объект с итогами по актам (для списка «объекты с деньгами», v0.3.5):
// строитель видит предварительный итог по каждому объекту, не открывая акты.
type ObjectBrief struct {
	Object
	Acts   int
	Total  float64
	Paid   float64
	Photos int // фото объекта, для кнопки «📷 показать» (v0.3.7)
}

// Debt — сколько осталось оплатить по объекту.
func (o ObjectBrief) Debt() float64 { return o.Total - o.Paid }

// CatalogItem — позиция прайс-листа (цены мастера, v0.3).
// Name хранится нормализованным (нижний регистр, ё→е, без двойных пробелов) —
// чтобы «Штукатурка» и «штукатурка» были одной позицией.
type CatalogItem struct {
	Name  string
	Unit  string
	Price float64
}

// DraftLine — позиция в черновике (FSM), сериализуется в JSON.
type DraftLine struct {
	Name  string  `json:"name"`
	Qty   float64 `json:"qty"`
	Unit  string  `json:"unit,omitempty"`
	Price float64 `json:"price"`
	Sum   float64 `json:"sum"`
}
