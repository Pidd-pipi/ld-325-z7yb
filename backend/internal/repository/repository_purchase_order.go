package repository

import (
	"errors"
	"fmt"

	"github.com/blueship581/cybuildprice/backend/internal/model"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"gorm.io/gorm"
)

type PurchaseOrderRepository interface {
	Create(model.PurchaseOrder) (model.PurchaseOrder, error)
	GetByID(uint) (model.PurchaseOrder, error)
	ListByUser(string, string) ([]model.PurchaseOrder, error)
	Save(model.PurchaseOrder) (model.PurchaseOrder, error)
	RegisterDelivery(model.PurchaseOrder, model.Delivery) (model.PurchaseOrder, error)
}
type purchaseOrderRepository struct{ db *gorm.DB }

func NewPurchaseOrderRepository(db *gorm.DB) PurchaseOrderRepository {
	return &purchaseOrderRepository{db}
}
func (r *purchaseOrderRepository) Create(order model.PurchaseOrder) (model.PurchaseOrder, error) {
	if err := r.db.Create(&order).Error; err != nil {
		return order, fmt.Errorf("create purchase order: %w", err)
	}
	return r.GetByID(order.ID)
}
func (r *purchaseOrderRepository) GetByID(id uint) (model.PurchaseOrder, error) {
	var order model.PurchaseOrder
	if err := r.withDetails(r.db).First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return order, apperrors.ErrNotFound
		}
		return order, fmt.Errorf("find purchase order: %w", err)
	}
	return order, nil
}
func (r *purchaseOrderRepository) ListByUser(userID, status string) ([]model.PurchaseOrder, error) {
	var orders []model.PurchaseOrder
	query := r.withDetails(r.db).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("purchase_orders.created_at DESC").Find(&orders).Error; err != nil {
		return nil, fmt.Errorf("list purchase orders: %w", err)
	}
	return orders, nil
}
func (r *purchaseOrderRepository) Save(order model.PurchaseOrder) (model.PurchaseOrder, error) {
	if err := r.db.Save(&order).Error; err != nil {
		return order, fmt.Errorf("save purchase order: %w", err)
	}
	return r.GetByID(order.ID)
}

// RegisterDelivery persists one arrival and the updated order atomically, and
// re-checks the cumulative cap inside the transaction so concurrent arrivals
// can never overshoot the ordered quantity.
func (r *purchaseOrderRepository) RegisterDelivery(order model.PurchaseOrder, delivery model.Delivery) (model.PurchaseOrder, error) {
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		var current model.PurchaseOrder
		if err := tx.First(&current, order.ID).Error; err != nil {
			return fmt.Errorf("reload purchase order: %w", err)
		}
		if current.ReceivedTotal+delivery.Quantity > current.Quantity {
			return apperrors.ErrExceedsOrdered
		}
		if err := tx.Create(&delivery).Error; err != nil {
			return fmt.Errorf("create delivery: %w", err)
		}
		if err := tx.Save(&order).Error; err != nil {
			return fmt.Errorf("update purchase order: %w", err)
		}
		return nil
	}); err != nil {
		return order, err
	}
	return r.GetByID(order.ID)
}
func (r *purchaseOrderRepository) withDetails(db *gorm.DB) *gorm.DB {
	return db.Preload("Product").Preload("Supplier").Preload("Deliveries", func(tx *gorm.DB) *gorm.DB {
		return tx.Order("deliveries.created_at ASC")
	})
}
