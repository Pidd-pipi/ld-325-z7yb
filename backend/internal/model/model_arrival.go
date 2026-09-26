package model

import (
	"time"

	"gorm.io/gorm"
)

// Arrival 记录采购单接单后的每一次分次到货。
type Arrival struct {
	gorm.Model
	PurchaseOrderID uint `gorm:"index"`
	Quantity        int
	Note            string
	ArrivedAt       time.Time
}
