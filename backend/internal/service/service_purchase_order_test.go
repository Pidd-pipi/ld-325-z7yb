package service

import (
	"testing"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
)

type mockOrderRepo struct{ order model.PurchaseOrder }

func (m *mockOrderRepo) Create(order model.PurchaseOrder) (model.PurchaseOrder, error) {
	order.ID = 1
	m.order = order
	return order, nil
}
func (m *mockOrderRepo) GetByID(uint) (model.PurchaseOrder, error) { return m.order, nil }
func (m *mockOrderRepo) ListByUser(string, string) ([]model.PurchaseOrder, error) {
	return []model.PurchaseOrder{m.order}, nil
}
func (m *mockOrderRepo) Save(order model.PurchaseOrder) (model.PurchaseOrder, error) {
	m.order = order
	return order, nil
}
func (m *mockOrderRepo) RegisterDelivery(order model.PurchaseOrder, _ model.Delivery) (model.PurchaseOrder, error) {
	m.order = order
	return order, nil
}

type mockOfferRepo struct{ offer model.Offer }

func (m mockOfferRepo) ListByProduct(uint) ([]model.Offer, error)        { return nil, nil }
func (m mockOfferRepo) GetByID(uint) (model.Offer, error)                { return m.offer, nil }
func (m mockOfferRepo) UpdateStatus(uint, string) (model.Offer, error)   { return model.Offer{}, nil }

func newOrderService(order model.PurchaseOrder, offer model.Offer) *PurchaseOrderService {
	return NewPurchaseOrderService(&mockOrderRepo{order}, mockOfferRepo{offer})
}

func testOffer() model.Offer {
	return model.Offer{ProductID: 3, SupplierID: 2, UnitPrice: 188, MOQ: 20, Freight: "满 100 ㎡免运费", DeliveryDays: 5}
}

func TestCreateValidatesMOQ(t *testing.T) {
	svc := newOrderService(model.PurchaseOrder{}, testOffer())
	_, err := svc.Create("demo-user", dto.CreatePurchaseOrderRequest{OfferID: 1, Quantity: 10})
	if err == nil {
		t.Fatal("expected MOQ validation error")
	}
	if _, ok := err.(*apperrors.BusinessError); !ok {
		t.Fatalf("expected business error, got %T", err)
	}
}

func TestCreateLocksOfferTerms(t *testing.T) {
	svc := newOrderService(model.PurchaseOrder{}, testOffer())
	view, err := svc.Create("demo-user", dto.CreatePurchaseOrderRequest{OfferID: 1, Quantity: 40})
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != constants.OrderStatusPending || view.SupplierID != 2 || view.UnitPrice != 188 || view.Freight != "满 100 ㎡免运费" {
		t.Fatalf("terms not locked: %+v", view)
	}
	if days := time.Until(view.ExpectedArrival).Hours() / 24; days < 4 || days > 5 {
		t.Fatalf("expected arrival not locked to delivery days: %v", days)
	}
	if view.TotalAmount != 188*40 {
		t.Fatalf("unexpected total amount: %v", view.TotalAmount)
	}
}

func pendingOrder() model.PurchaseOrder {
	return model.PurchaseOrder{UserID: "demo-user", Quantity: 20, UnitPrice: 100, Status: constants.OrderStatusPending, ExpectedArrival: time.Now().AddDate(0, 0, 3)}
}

func TestStatusTransitions(t *testing.T) {
	t.Run("accept then cancel rejected", func(t *testing.T) {
		svc := newOrderService(pendingOrder(), testOffer())
		accepted, err := svc.Accept(1)
		if err != nil || accepted.Status != constants.OrderStatusInTransit {
			t.Fatalf("accept failed: %v %v", accepted.Status, err)
		}
		if _, err = svc.Cancel("demo-user", 1); err == nil {
			t.Fatal("cancel after accept must fail")
		}
	})
	t.Run("cancel only before accept", func(t *testing.T) {
		svc := newOrderService(pendingOrder(), testOffer())
		cancelled, err := svc.Cancel("demo-user", 1)
		if err != nil || cancelled.Status != constants.OrderStatusCancelled {
			t.Fatalf("cancel failed: %v %v", cancelled.Status, err)
		}
		if _, err = svc.Accept(1); err == nil {
			t.Fatal("accept after cancel must fail")
		}
	})
	t.Run("reject keeps reason", func(t *testing.T) {
		svc := newOrderService(pendingOrder(), testOffer())
		rejected, err := svc.Reject(1, "厂家停产")
		if err != nil || rejected.Status != constants.OrderStatusRejected || rejected.RejectReason != "厂家停产" {
			t.Fatalf("reject failed: %+v %v", rejected, err)
		}
	})
	t.Run("foreign user cannot cancel", func(t *testing.T) {
		svc := newOrderService(pendingOrder(), testOffer())
		if _, err := svc.Cancel("other-user", 1); err == nil {
			t.Fatal("expected ownership error")
		}
	})
}

func TestReceiveLifecycle(t *testing.T) {
	inTransit := pendingOrder()
	inTransit.Status = constants.OrderStatusInTransit
	t.Run("partial then completed", func(t *testing.T) {
		svc := newOrderService(inTransit, testOffer())
		partial, err := svc.Receive("demo-user", 1, dto.ReceiveDeliveryRequest{Quantity: 8})
		if err != nil || partial.Status != constants.OrderStatusPartial || partial.ReceivedTotal != 8 {
			t.Fatalf("partial failed: %+v %v", partial, err)
		}
		done, err := svc.Receive("demo-user", 1, dto.ReceiveDeliveryRequest{Quantity: 12})
		if err != nil || done.Status != constants.OrderStatusCompleted || done.ReceivedTotal != 20 {
			t.Fatalf("complete failed: %+v %v", done, err)
		}
	})
	t.Run("cumulative cannot exceed ordered", func(t *testing.T) {
		svc := newOrderService(inTransit, testOffer())
		if _, err := svc.Receive("demo-user", 1, dto.ReceiveDeliveryRequest{Quantity: 21}); err == nil {
			t.Fatal("expected exceed error")
		}
	})
	t.Run("pending order cannot receive", func(t *testing.T) {
		svc := newOrderService(pendingOrder(), testOffer())
		if _, err := svc.Receive("demo-user", 1, dto.ReceiveDeliveryRequest{Quantity: 1}); err == nil {
			t.Fatal("expected transition error")
		}
	})
}

func TestDelayDays(t *testing.T) {
	overdue := pendingOrder()
	overdue.Status = constants.OrderStatusInTransit
	overdue.ExpectedArrival = time.Now().AddDate(0, 0, -3)
	svc := newOrderService(overdue, testOffer())
	views, err := svc.List("demo-user", "")
	if err != nil || len(views) != 1 {
		t.Fatalf("list failed: %v", err)
	}
	if views[0].DelayDays != 3 {
		t.Fatalf("expected 3 delay days, got %d", views[0].DelayDays)
	}
	overdue.Status = constants.OrderStatusCompleted
	views, _ = newOrderService(overdue, testOffer()).List("demo-user", "")
	if views[0].DelayDays != 0 {
		t.Fatalf("completed order must not be overdue, got %d", views[0].DelayDays)
	}
}
