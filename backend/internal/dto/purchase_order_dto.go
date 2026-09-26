package dto

import "github.com/blueship581/cybuildprice/backend/internal/model"

type CreatePurchaseOrderRequest struct {
	OfferID  uint `json:"offer_id" validate:"required,gt=0"`
	Quantity int  `json:"quantity" validate:"required,gt=0"`
}
type RejectPurchaseOrderRequest struct {
	Reason string `json:"reason" validate:"required,min=2,max=200"`
}
type ReceiveDeliveryRequest struct {
	Quantity int    `json:"quantity" validate:"required,gt=0"`
	Note     string `json:"note" validate:"max=120"`
}

// PurchaseOrderView flattens the stored order with computed display fields.
type PurchaseOrderView struct {
	model.PurchaseOrder
	TotalAmount float64
	DelayDays   int
}
