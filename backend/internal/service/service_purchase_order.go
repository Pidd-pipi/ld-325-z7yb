package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
	"github.com/google/uuid"
)

type PurchaseOrderService struct {
	orders repository.PurchaseOrderRepository
	offers repository.OfferRepository
}

func NewPurchaseOrderService(orders repository.PurchaseOrderRepository, offers repository.OfferRepository) *PurchaseOrderService {
	return &PurchaseOrderService{orders, offers}
}

// Create locks supplier, unit price, freight and the expected arrival date
// (order date + promised delivery days) into a new pending order.
func (s *PurchaseOrderService) Create(userID string, input dto.CreatePurchaseOrderRequest) (dto.PurchaseOrderView, error) {
	offer, err := s.offers.GetByID(input.OfferID)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if input.Quantity < offer.MOQ {
		return dto.PurchaseOrderView{}, apperrors.NewBusiness(constants.ErrorBusiness, fmt.Sprintf("数量不能低于起订量 %d", offer.MOQ))
	}
	now := time.Now()
	order := model.PurchaseOrder{
		OrderNo:         orderNo(now),
		UserID:          userID,
		OfferID:         offer.ID,
		ProductID:       offer.ProductID,
		SupplierID:      offer.SupplierID,
		Quantity:        input.Quantity,
		UnitPrice:       offer.UnitPrice,
		Freight:         offer.Freight,
		ExpectedArrival: now.AddDate(0, 0, offer.DeliveryDays),
		Status:          constants.OrderStatusPending,
	}
	saved, err := s.orders.Create(order)
	if err != nil {
		return dto.PurchaseOrderView{}, fmt.Errorf("create purchase order service: %w", err)
	}
	return toPurchaseOrderView(saved, now), nil
}

func (s *PurchaseOrderService) List(userID, status string) ([]dto.PurchaseOrderView, error) {
	rows, err := s.orders.ListByUser(userID, status)
	if err != nil {
		return nil, fmt.Errorf("list purchase orders service: %w", err)
	}
	now := time.Now()
	views := make([]dto.PurchaseOrderView, 0, len(rows))
	for _, order := range rows {
		views = append(views, toPurchaseOrderView(order, now))
	}
	return views, nil
}

// Accept moves a pending order into transit; only pending orders can be accepted.
func (s *PurchaseOrderService) Accept(id uint) (dto.PurchaseOrderView, error) {
	order, err := s.orders.GetByID(id)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if order.Status != constants.OrderStatusPending {
		return dto.PurchaseOrderView{}, apperrors.NewBusiness(constants.ErrorBusiness, "只有待接单的采购单才能接单")
	}
	now := time.Now()
	order.Status = constants.OrderStatusInTransit
	order.RespondedAt = &now
	return s.save(order)
}

// Reject keeps the supplier's reason for being unable to supply.
func (s *PurchaseOrderService) Reject(id uint, reason string) (dto.PurchaseOrderView, error) {
	order, err := s.orders.GetByID(id)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if order.Status != constants.OrderStatusPending {
		return dto.PurchaseOrderView{}, apperrors.NewBusiness(constants.ErrorBusiness, "只有待接单的采购单才能标记无法供货")
	}
	now := time.Now()
	order.Status = constants.OrderStatusRejected
	order.RejectReason = reason
	order.RespondedAt = &now
	return s.save(order)
}

// Cancel is only allowed before the supplier accepts the order.
func (s *PurchaseOrderService) Cancel(userID string, id uint) (dto.PurchaseOrderView, error) {
	order, err := s.ownedOrder(userID, id)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if order.Status != constants.OrderStatusPending {
		return dto.PurchaseOrderView{}, apperrors.NewBusiness(constants.ErrorBusiness, "商家接单后不可取消，仅待接单阶段可取消")
	}
	order.Status = constants.OrderStatusCancelled
	return s.save(order)
}

// Receive registers one partial arrival; the order completes only when the
// cumulative arrivals equal the ordered quantity.
func (s *PurchaseOrderService) Receive(userID string, id uint, input dto.ReceiveDeliveryRequest) (dto.PurchaseOrderView, error) {
	order, err := s.ownedOrder(userID, id)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if order.Status != constants.OrderStatusInTransit && order.Status != constants.OrderStatusPartial {
		return dto.PurchaseOrderView{}, apperrors.NewBusiness(constants.ErrorBusiness, "商家接单后才能登记到货")
	}
	if order.ReceivedTotal+input.Quantity > order.Quantity {
		return dto.PurchaseOrderView{}, apperrors.NewBusiness(constants.ErrorBusiness, fmt.Sprintf("累计到货不能超过下单量 %d", order.Quantity))
	}
	order.ReceivedTotal += input.Quantity
	if order.ReceivedTotal == order.Quantity {
		now := time.Now()
		order.Status = constants.OrderStatusCompleted
		order.CompletedAt = &now
	} else {
		order.Status = constants.OrderStatusPartial
	}
	delivery := model.Delivery{PurchaseOrderID: order.ID, Quantity: input.Quantity, Note: input.Note}
	saved, err := s.orders.RegisterDelivery(order, delivery)
	if err != nil {
		if errors.Is(err, apperrors.ErrExceedsOrdered) {
			return dto.PurchaseOrderView{}, apperrors.NewBusiness(constants.ErrorBusiness, fmt.Sprintf("累计到货不能超过下单量 %d", order.Quantity))
		}
		return dto.PurchaseOrderView{}, fmt.Errorf("register delivery service: %w", err)
	}
	return toPurchaseOrderView(saved, time.Now()), nil
}

func (s *PurchaseOrderService) ownedOrder(userID string, id uint) (model.PurchaseOrder, error) {
	order, err := s.orders.GetByID(id)
	if err != nil {
		return order, err
	}
	if order.UserID != userID {
		return order, apperrors.ErrUnauthorized
	}
	return order, nil
}

func (s *PurchaseOrderService) save(order model.PurchaseOrder) (dto.PurchaseOrderView, error) {
	saved, err := s.orders.Save(order)
	if err != nil {
		return dto.PurchaseOrderView{}, fmt.Errorf("save purchase order service: %w", err)
	}
	return toPurchaseOrderView(saved, time.Now()), nil
}

func toPurchaseOrderView(order model.PurchaseOrder, now time.Time) dto.PurchaseOrderView {
	view := dto.PurchaseOrderView{PurchaseOrder: order, TotalAmount: order.UnitPrice * float64(order.Quantity)}
	if constants.OrderIsOpen(order.Status) && now.After(order.ExpectedArrival) {
		view.DelayDays = int(now.Sub(order.ExpectedArrival).Hours() / 24)
	}
	return view
}

func orderNo(now time.Time) string {
	return constants.OrderNoPrefix + now.Format("20060102") + "-" + strings.ToUpper(uuid.NewString()[:8])
}
