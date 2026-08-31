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

// PhotoRec — фото скрытых работ (привязка к акту или объекту).
type PhotoRec struct {
	ID       int64
	ActID    *int64
	ObjectID int64
	FileID   string
	FilePath string
	Caption  string
	At       time.Time
}

// DraftLine — позиция в черновике (FSM), сериализуется в JSON.
type DraftLine struct {
	Name  string  `json:"name"`
	Qty   float64 `json:"qty"`
	Unit  string  `json:"unit,omitempty"`
	Price float64 `json:"price"`
	Sum   float64 `json:"sum"`
}
