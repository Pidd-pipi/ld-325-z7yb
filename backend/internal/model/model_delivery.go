package model

import "gorm.io/gorm"

// Delivery records one partial arrival against a purchase order.
type Delivery struct {
	gorm.Model
	PurchaseOrderID uint `gorm:"index"`
	Quantity        int
	Note            string
}
