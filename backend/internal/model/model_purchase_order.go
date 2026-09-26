package model

import (
	"time"

	"gorm.io/gorm"
)

// PurchaseOrder locks the commercial terms of an offer at order time so later
// price or freight changes never affect an in-flight order.
type PurchaseOrder struct {
	gorm.Model
	OrderNo         string `gorm:"uniqueIndex;not null"`
	UserID          string `gorm:"index"`
	OfferID         uint
	ProductID       uint
	Product         Product
	SupplierID      uint
	Supplier        Supplier
	Quantity        int
	UnitPrice       float64
	Freight         string
	ExpectedArrival time.Time
	Status          string `gorm:"index"`
	RejectReason    string
	ReceivedTotal   int
	RespondedAt     *time.Time
	CompletedAt     *time.Time
	Deliveries      []Delivery
}
