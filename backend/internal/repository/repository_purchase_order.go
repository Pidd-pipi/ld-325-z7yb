package repository

import (
	"fmt"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/gorm"
)

type PurchaseOrderRepository interface {
	Create(model.PurchaseOrder) (model.PurchaseOrder, error)
	ListByUser(string) ([]model.PurchaseOrder, error)
	Get(uint) (model.PurchaseOrder, error)
	Save(model.PurchaseOrder) (model.PurchaseOrder, error)
	RegisterArrival(uint, model.Arrival) (model.PurchaseOrder, error)
}
type purchaseOrderRepository struct{ db *gorm.DB }

func NewPurchaseOrderRepository(db *gorm.DB) PurchaseOrderRepository {
	return &purchaseOrderRepository{db}
}
func (r *purchaseOrderRepository) Create(value model.PurchaseOrder) (model.PurchaseOrder, error) {
	if err := r.db.Create(&value).Error; err != nil {
		return value, fmt.Errorf("create purchase order: %w", err)
	}
	return value, nil
}
func (r *purchaseOrderRepository) ListByUser(user string) ([]model.PurchaseOrder, error) {
	var values []model.PurchaseOrder
	if err := r.db.Preload("Product").Preload("Arrivals").Where("user_id = ?", user).Order("created_at DESC").Find(&values).Error; err != nil {
		return nil, fmt.Errorf("list purchase orders: %w", err)
	}
	return values, nil
}
func (r *purchaseOrderRepository) Get(id uint) (model.PurchaseOrder, error) {
	var value model.PurchaseOrder
	err := r.db.Preload("Product").Preload("Arrivals").First(&value, id).Error
	if err == gorm.ErrRecordNotFound {
		return value, apperrors.ErrNotFound
	}
	if err != nil {
		return value, fmt.Errorf("get purchase order: %w", err)
	}
	return value, nil
}

// Save 只回写流转过程中允许变化的字段，避免触碰下单时锁定的条款。
func (r *purchaseOrderRepository) Save(value model.PurchaseOrder) (model.PurchaseOrder, error) {
	if err := r.db.Select("Status", "RejectReason", "ReceivedQuantity").Save(&value).Error; err != nil {
		return value, fmt.Errorf("save purchase order: %w", err)
	}
	return value, nil
}

// RegisterArrival 在事务内复核状态，并用带守卫条件的原子更新保证累计到货不会超过下单量。
func (r *purchaseOrderRepository) RegisterArrival(id uint, arrival model.Arrival) (model.PurchaseOrder, error) {
	var order model.PurchaseOrder
	err := r.db.Transaction(func(tx *gorm.DB) error {
		err := tx.First(&order, id).Error
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("lock purchase order: %w", err)
		}
		if order.Status != constants.OrderStatusInTransit && order.Status != constants.OrderStatusPartial {
			return apperrors.ErrInvalidTransition
		}
		if arrival.Quantity <= 0 || order.ReceivedQuantity+arrival.Quantity > order.Quantity {
			return apperrors.ErrArrivalExceeds
		}
		guarded := tx.Model(&model.PurchaseOrder{}).
			Where("id = ? AND status IN ? AND received_quantity + ? <= quantity", id, []string{constants.OrderStatusInTransit, constants.OrderStatusPartial}, arrival.Quantity).
			Updates(map[string]any{
				"received_quantity": gorm.Expr("received_quantity + ?", arrival.Quantity),
				"status":            gorm.Expr("CASE WHEN received_quantity + ? >= quantity THEN ? ELSE ? END", arrival.Quantity, constants.OrderStatusCompleted, constants.OrderStatusPartial),
			})
		if guarded.Error != nil {
			return fmt.Errorf("update purchase order arrival: %w", guarded.Error)
		}
		if guarded.RowsAffected == 0 {
			return apperrors.ErrArrivalExceeds
		}
		arrival.PurchaseOrderID = order.ID
		if err := tx.Create(&arrival).Error; err != nil {
			return fmt.Errorf("create arrival: %w", err)
		}
		return nil
	})
	if err != nil {
		return order, err
	}
	return r.Get(id)
}
