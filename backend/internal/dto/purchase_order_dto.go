package dto

import "time"

type CreatePurchaseOrderRequest struct {
	OfferID  uint `json:"offer_id" validate:"required,gt=0"`
	Quantity int  `json:"quantity" validate:"required,gt=0,lte=1000000"`
}

type RejectPurchaseOrderRequest struct {
	Reason string `json:"reason" validate:"required,max=200"`
}

type RegisterArrivalRequest struct {
	Quantity int    `json:"quantity" validate:"required,gt=0,lte=1000000"`
	Note     string `json:"note" validate:"max=200"`
}

type ArrivalView struct {
	ID        uint      `json:"ID"`
	Quantity  int       `json:"Quantity"`
	Note      string    `json:"Note"`
	ArrivedAt time.Time `json:"ArrivedAt"`
}

// PurchaseOrderView 面向采购记录列表的读模型，DelayDays 由服务端按状态实时计算。
type PurchaseOrderView struct {
	ID               uint          `json:"ID"`
	Status           string        `json:"Status"`
	ProductName      string        `json:"ProductName"`
	ProductUnit      string        `json:"ProductUnit"`
	SupplierName     string        `json:"SupplierName"`
	UnitPrice        float64       `json:"UnitPrice"`
	Freight          string        `json:"Freight"`
	Quantity         int           `json:"Quantity"`
	ReceivedQuantity int           `json:"ReceivedQuantity"`
	ExpectedArrival  time.Time     `json:"ExpectedArrival"`
	RejectReason     string        `json:"RejectReason,omitempty"`
	DelayDays        int           `json:"DelayDays"`
	Arrivals         []ArrivalView `json:"Arrivals"`
	CreatedAt        time.Time     `json:"CreatedAt"`
}
