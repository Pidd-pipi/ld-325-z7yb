package constants

const (
	OrderStatusPending   = "pending"
	OrderStatusInTransit = "in_transit"
	OrderStatusPartial   = "partial"
	OrderStatusCompleted = "completed"
	OrderStatusRejected  = "rejected"
	OrderStatusCancelled = "cancelled"
	OrderNoPrefix        = "PO"
)

// OrderIsOpen reports whether the order still waits on supplier or arrivals.
func OrderIsOpen(status string) bool {
	return status == OrderStatusPending || status == OrderStatusInTransit || status == OrderStatusPartial
}

// ValidOrderStatus guards the list filter against unknown status values.
func ValidOrderStatus(status string) bool {
	return OrderIsOpen(status) || status == OrderStatusCompleted || status == OrderStatusRejected || status == OrderStatusCancelled
}
