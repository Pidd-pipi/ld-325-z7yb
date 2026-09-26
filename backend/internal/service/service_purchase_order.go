package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
)

type PurchaseOrderService struct {
	orders   repository.PurchaseOrderRepository
	offers   repository.OfferRepository
	products repository.ProductRepository
	logger   *slog.Logger
}

func NewPurchaseOrderService(orders repository.PurchaseOrderRepository, offers repository.OfferRepository, products repository.ProductRepository, logger *slog.Logger) *PurchaseOrderService {
	return &PurchaseOrderService{orders, offers, products, logger}
}

// Create 校验起订量后锁定商家、单价、运费与预计到货日，生成待接单采购单。
func (s *PurchaseOrderService) Create(user string, input dto.CreatePurchaseOrderRequest) (dto.PurchaseOrderView, error) {
	offer, err := s.offers.Get(input.OfferID)
	if err != nil {
		return dto.PurchaseOrderView{}, fmt.Errorf("load offer for purchase order: %w", err)
	}
	product, err := s.products.Get(offer.ProductID)
	if err != nil {
		return dto.PurchaseOrderView{}, fmt.Errorf("load product for purchase order: %w", err)
	}
	if input.Quantity < offer.MOQ {
		return dto.PurchaseOrderView{}, &apperrors.BusinessError{
			Code:    constants.ErrorBusiness,
			Message: fmt.Sprintf("起订量为 %d %s，请调整下单数量", offer.MOQ, product.Unit),
			Err:     apperrors.ErrBelowMOQ,
		}
	}
	order := model.PurchaseOrder{
		UserID:           user,
		OfferID:          offer.ID,
		ProductID:        product.ID,
		SupplierID:       offer.SupplierID,
		SupplierName:     offer.Supplier.Name,
		UnitPrice:        offer.UnitPrice,
		Freight:          offer.Freight,
		Quantity:         input.Quantity,
		ReceivedQuantity: 0,
		ExpectedArrival:  time.Now().AddDate(0, 0, offer.DeliveryDays),
		Status:           constants.OrderStatusPending,
	}
	created, err := s.orders.Create(order)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	created.Product = product
	s.logger.Info("purchase order created", "order_id", created.ID, "offer_id", offer.ID, "quantity", input.Quantity)
	return toPurchaseOrderView(created, time.Now()), nil
}

func (s *PurchaseOrderService) List(user string) ([]dto.PurchaseOrderView, error) {
	rows, err := s.orders.ListByUser(user)
	if err != nil {
		return nil, fmt.Errorf("list purchase orders service: %w", err)
	}
	now := time.Now()
	views := make([]dto.PurchaseOrderView, 0, len(rows))
	for _, row := range rows {
		views = append(views, toPurchaseOrderView(row, now))
	}
	return views, nil
}

// Accept 由商家把待接单采购单转为在途。
func (s *PurchaseOrderService) Accept(id uint) (dto.PurchaseOrderView, error) {
	order, err := s.orders.Get(id)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if order.Status != constants.OrderStatusPending {
		return dto.PurchaseOrderView{}, transitionError("只有待接单的采购单才能接单")
	}
	order.Status = constants.OrderStatusInTransit
	return s.save(order, "purchase order accepted")
}

// Reject 由商家说明无法供货并保留原因。
func (s *PurchaseOrderService) Reject(id uint, reason string) (dto.PurchaseOrderView, error) {
	order, err := s.orders.Get(id)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if order.Status != constants.OrderStatusPending {
		return dto.PurchaseOrderView{}, transitionError("只有待接单的采购单才能标记无法供货")
	}
	order.Status = constants.OrderStatusRejected
	order.RejectReason = reason
	return s.save(order, "purchase order rejected")
}

// Cancel 仅允许采购人在商家接单前取消。
func (s *PurchaseOrderService) Cancel(id uint, user string) (dto.PurchaseOrderView, error) {
	order, err := s.orders.Get(id)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if order.UserID != user {
		return dto.PurchaseOrderView{}, forbiddenError()
	}
	if order.Status != constants.OrderStatusPending {
		return dto.PurchaseOrderView{}, transitionError("商家接单后不能取消采购单")
	}
	order.Status = constants.OrderStatusCancelled
	return s.save(order, "purchase order cancelled")
}

// RegisterArrival 由采购人分次登记到货，累计到货不能超过下单量，全部到货才完成。
func (s *PurchaseOrderService) RegisterArrival(id uint, user string, input dto.RegisterArrivalRequest) (dto.PurchaseOrderView, error) {
	order, err := s.orders.Get(id)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	if order.UserID != user {
		return dto.PurchaseOrderView{}, forbiddenError()
	}
	if order.Status != constants.OrderStatusInTransit && order.Status != constants.OrderStatusPartial {
		return dto.PurchaseOrderView{}, transitionError("商家接单后才能登记到货")
	}
	remaining := order.Quantity - order.ReceivedQuantity
	if input.Quantity > remaining {
		return dto.PurchaseOrderView{}, &apperrors.BusinessError{
			Code:    constants.ErrorBusiness,
			Message: fmt.Sprintf("累计到货不能超过下单量，还可登记 %d %s", remaining, order.Product.Unit),
			Err:     apperrors.ErrArrivalExceeds,
		}
	}
	arrival := model.Arrival{Quantity: input.Quantity, Note: input.Note, ArrivedAt: time.Now()}
	updated, err := s.orders.RegisterArrival(id, arrival)
	if errors.Is(err, apperrors.ErrArrivalExceeds) {
		return dto.PurchaseOrderView{}, &apperrors.BusinessError{Code: constants.ErrorBusiness, Message: "累计到货不能超过下单量", Err: err}
	}
	if errors.Is(err, apperrors.ErrInvalidTransition) {
		return dto.PurchaseOrderView{}, transitionError("当前状态不能登记到货")
	}
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	s.logger.Info("purchase order arrival registered", "order_id", id, "quantity", input.Quantity, "status", updated.Status)
	return toPurchaseOrderView(updated, time.Now()), nil
}

func (s *PurchaseOrderService) save(order model.PurchaseOrder, event string) (dto.PurchaseOrderView, error) {
	saved, err := s.orders.Save(order)
	if err != nil {
		return dto.PurchaseOrderView{}, err
	}
	s.logger.Info(event, "order_id", order.ID)
	return toPurchaseOrderView(saved, time.Now()), nil
}

func transitionError(message string) error {
	return &apperrors.BusinessError{Code: constants.ErrorBusiness, Message: message, Err: apperrors.ErrInvalidTransition}
}

func forbiddenError() error {
	return &apperrors.BusinessError{Code: constants.ErrorForbidden, Message: "只能操作自己的采购单", Err: apperrors.ErrForbidden}
}

func toPurchaseOrderView(order model.PurchaseOrder, now time.Time) dto.PurchaseOrderView {
	arrivals := make([]dto.ArrivalView, 0, len(order.Arrivals))
	var lastArrival time.Time
	for _, item := range order.Arrivals {
		arrivals = append(arrivals, dto.ArrivalView{ID: item.ID, Quantity: item.Quantity, Note: item.Note, ArrivedAt: item.ArrivedAt})
		if item.ArrivedAt.After(lastArrival) {
			lastArrival = item.ArrivedAt
		}
	}
	delay := 0
	switch order.Status {
	case constants.OrderStatusInTransit, constants.OrderStatusPartial:
		delay = overdueDays(order.ExpectedArrival, now)
	case constants.OrderStatusCompleted:
		delay = overdueDays(order.ExpectedArrival, lastArrival)
	}
	return dto.PurchaseOrderView{
		ID:               order.ID,
		Status:           order.Status,
		ProductName:      order.Product.Name,
		ProductUnit:      order.Product.Unit,
		SupplierName:     order.SupplierName,
		UnitPrice:        order.UnitPrice,
		Freight:          order.Freight,
		Quantity:         order.Quantity,
		ReceivedQuantity: order.ReceivedQuantity,
		ExpectedArrival:  order.ExpectedArrival,
		RejectReason:     order.RejectReason,
		DelayDays:        delay,
		Arrivals:         arrivals,
		CreatedAt:        order.CreatedAt,
	}
}

// overdueDays 按自然日计算 ref 相对 expected 的延迟天数，未逾期返回 0。
func overdueDays(expected, ref time.Time) int {
	if ref.IsZero() {
		return 0
	}
	location := expected.Location()
	start := expected.In(location)
	end := ref.In(location)
	startDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
	endDate := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, location)
	days := int(endDate.Sub(startDate).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
