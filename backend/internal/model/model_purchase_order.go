package model

import (
	"time"

	"gorm.io/gorm"
)

// PurchaseOrder 在下单时锁定商家、单价、运费与预计到货日，后续流转不再回读报价。
type PurchaseOrder struct {
	gorm.Model
	UserID           string `gorm:"index"`
	OfferID          uint
	ProductID        uint
	Product          Product
	SupplierID       uint
	SupplierName     string
	UnitPrice        float64
	Freight          string
	Quantity         int
	ReceivedQuantity int
	ExpectedArrival  time.Time
	Status           string `gorm:"index"`
	RejectReason     string
	Arrivals         []Arrival
}
