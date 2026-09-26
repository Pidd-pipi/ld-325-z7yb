package service

import (
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"github.com/blueship581/cybuildprice/backend/internal/repository"
	"gorm.io/gorm"
)

type mockPurchaseOrderRepo struct {
	order  model.PurchaseOrder
	orders []model.PurchaseOrder
}

func (m *mockPurchaseOrderRepo) Create(v model.PurchaseOrder) (model.PurchaseOrder, error) {
	v.ID = 7
	return v, nil
}
func (m *mockPurchaseOrderRepo) ListByUser(string) ([]model.PurchaseOrder, error) {
	return m.orders, nil
}
func (m *mockPurchaseOrderRepo) Get(uint) (model.PurchaseOrder, error) { return m.order, nil }
func (m *mockPurchaseOrderRepo) Save(v model.PurchaseOrder) (model.PurchaseOrder, error) {
	m.order = v
	return v, nil
}
func (m *mockPurchaseOrderRepo) RegisterArrival(_ uint, a model.Arrival) (model.PurchaseOrder, error) {
	m.order.ReceivedQuantity += a.Quantity
	m.order.Arrivals = append(m.order.Arrivals, a)
	m.order.Status = constants.OrderStatusPartial
	if m.order.ReceivedQuantity >= m.order.Quantity {
		m.order.Status = constants.OrderStatusCompleted
	}
	return m.order, nil
}

type mockOfferRepo struct{ offer model.Offer }

func (m mockOfferRepo) ListByProduct(uint) ([]model.Offer, error)      { return nil, nil }
func (m mockOfferRepo) Get(uint) (model.Offer, error)                  { return m.offer, nil }
func (m mockOfferRepo) UpdateStatus(uint, string) (model.Offer, error) { return model.Offer{}, nil }

type stubProductRepo struct{ product model.Product }

func (m stubProductRepo) List(string, string, string, int, int) ([]model.Product, int64, error) {
	return nil, 0, nil
}
func (m stubProductRepo) Get(uint) (model.Product, error)         { return m.product, nil }
func (m stubProductRepo) Compare([]uint) ([]model.Product, error) { return nil, nil }

func newPurchaseOrderService(orders repository.PurchaseOrderRepository, offers repository.OfferRepository, products repository.ProductRepository) *PurchaseOrderService {
	return NewPurchaseOrderService(orders, offers, products, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func orderFixtures() (repository.OfferRepository, repository.ProductRepository) {
	offers := mockOfferRepo{offer: model.Offer{
		UnitPrice: 398, MOQ: 10, Freight: "满 30 片送货上门", DeliveryDays: 3,
		SupplierID: 2, Supplier: model.Supplier{Name: "筑家优选旗舰店"},
	}}
	products := stubProductRepo{product: model.Product{Name: "云纹岩板 900×1800", Unit: "片"}}
	return offers, products
}

func TestPurchaseOrderCreateValidatesMOQ(t *testing.T) {
	offers, products := orderFixtures()
	svc := newPurchaseOrderService(&mockPurchaseOrderRepo{}, offers, products)
	cases := []struct {
		name     string
		quantity int
		wantErr  bool
	}{
		{"below moq rejected", 5, true},
		{"equal to moq accepted", 10, false},
		{"above moq accepted", 30, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create("demo-user", dto.CreatePurchaseOrderRequest{OfferID: 1, Quantity: tc.quantity})
			if tc.wantErr && err == nil {
				t.Fatal("expected MOQ validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPurchaseOrderCreateLocksTerms(t *testing.T) {
	offers, products := orderFixtures()
	svc := newPurchaseOrderService(&mockPurchaseOrderRepo{}, offers, products)
	view, err := svc.Create("demo-user", dto.CreatePurchaseOrderRequest{OfferID: 1, Quantity: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Status != constants.OrderStatusPending {
		t.Fatalf("expected pending status, got %s", view.Status)
	}
	if view.SupplierName != "筑家优选旗舰店" || view.UnitPrice != 398 || view.Freight != "满 30 片送货上门" {
		t.Fatalf("locked terms mismatch: %+v", view)
	}
	expected := time.Now().AddDate(0, 0, 3)
	if view.ExpectedArrival.YearDay() != expected.YearDay() {
		t.Fatalf("expected arrival around %v, got %v", expected, view.ExpectedArrival)
	}
	if view.ProductName == "" || view.ProductUnit != "片" {
		t.Fatalf("product snapshot missing: %+v", view)
	}
}

func TestPurchaseOrderAcceptOnlyFromPending(t *testing.T) {
	offers, products := orderFixtures()
	repo := &mockPurchaseOrderRepo{order: model.PurchaseOrder{UserID: "demo-user", Status: constants.OrderStatusPending}}
	svc := newPurchaseOrderService(repo, offers, products)
	view, err := svc.Accept(1)
	if err != nil || view.Status != constants.OrderStatusInTransit {
		t.Fatalf("accept failed: %v %s", err, view.Status)
	}
	if _, err = svc.Accept(1); err == nil {
		t.Fatal("expected error accepting an in-transit order")
	}
}

func TestPurchaseOrderRejectKeepsReason(t *testing.T) {
	offers, products := orderFixtures()
	repo := &mockPurchaseOrderRepo{order: model.PurchaseOrder{UserID: "demo-user", Status: constants.OrderStatusPending}}
	svc := newPurchaseOrderService(repo, offers, products)
	view, err := svc.Reject(1, "岩板缺货，预计 20 天后恢复")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Status != constants.OrderStatusRejected || view.RejectReason != "岩板缺货，预计 20 天后恢复" {
		t.Fatalf("reject reason not preserved: %+v", view)
	}
}

func TestPurchaseOrderCancelOnlyBeforeAccept(t *testing.T) {
	offers, products := orderFixtures()
	cases := []struct {
		name    string
		status  string
		user    string
		wantErr bool
	}{
		{"owner cancels pending", constants.OrderStatusPending, "demo-user", false},
		{"other user cannot cancel", constants.OrderStatusPending, "someone-else", true},
		{"accepted order cannot cancel", constants.OrderStatusInTransit, "demo-user", true},
		{"completed order cannot cancel", constants.OrderStatusCompleted, "demo-user", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockPurchaseOrderRepo{order: model.PurchaseOrder{UserID: "demo-user", Status: tc.status}}
			svc := newPurchaseOrderService(repo, offers, products)
			view, err := svc.Cancel(1, tc.user)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected cancel to be rejected")
				}
				return
			}
			if err != nil || view.Status != constants.OrderStatusCancelled {
				t.Fatalf("cancel failed: %v %s", err, view.Status)
			}
		})
	}
}

func TestPurchaseOrderArrivalRules(t *testing.T) {
	offers, products := orderFixtures()
	t.Run("pending order cannot register arrival", func(t *testing.T) {
		repo := &mockPurchaseOrderRepo{order: model.PurchaseOrder{UserID: "demo-user", Status: constants.OrderStatusPending, Quantity: 10}}
		svc := newPurchaseOrderService(repo, offers, products)
		if _, err := svc.RegisterArrival(1, "demo-user", dto.RegisterArrivalRequest{Quantity: 2}); err == nil {
			t.Fatal("expected transition error")
		}
	})
	t.Run("cumulative arrivals cannot exceed ordered quantity", func(t *testing.T) {
		repo := &mockPurchaseOrderRepo{order: model.PurchaseOrder{UserID: "demo-user", Status: constants.OrderStatusPartial, Quantity: 10, ReceivedQuantity: 8, Product: model.Product{Unit: "片"}}}
		svc := newPurchaseOrderService(repo, offers, products)
		_, err := svc.RegisterArrival(1, "demo-user", dto.RegisterArrivalRequest{Quantity: 5})
		if err == nil || !strings.Contains(err.Error(), "还可登记 2 片") {
			t.Fatalf("expected remaining hint, got %v", err)
		}
	})
	t.Run("full arrival completes the order", func(t *testing.T) {
		repo := &mockPurchaseOrderRepo{order: model.PurchaseOrder{UserID: "demo-user", Status: constants.OrderStatusInTransit, Quantity: 10}}
		svc := newPurchaseOrderService(repo, offers, products)
		view, err := svc.RegisterArrival(1, "demo-user", dto.RegisterArrivalRequest{Quantity: 4})
		if err != nil || view.Status != constants.OrderStatusPartial || view.ReceivedQuantity != 4 {
			t.Fatalf("partial arrival failed: %v %+v", err, view)
		}
		view, err = svc.RegisterArrival(1, "demo-user", dto.RegisterArrivalRequest{Quantity: 6})
		if err != nil || view.Status != constants.OrderStatusCompleted {
			t.Fatalf("completion failed: %v %+v", err, view)
		}
	})
	t.Run("only owner can register", func(t *testing.T) {
		repo := &mockPurchaseOrderRepo{order: model.PurchaseOrder{UserID: "demo-user", Status: constants.OrderStatusInTransit, Quantity: 10}}
		svc := newPurchaseOrderService(repo, offers, products)
		if _, err := svc.RegisterArrival(1, "someone-else", dto.RegisterArrivalRequest{Quantity: 1}); err == nil {
			t.Fatal("expected forbidden error")
		}
	})
}

func TestPurchaseOrderDelayDays(t *testing.T) {
	offers, products := orderFixtures()
	now := time.Now()
	repo := &mockPurchaseOrderRepo{orders: []model.PurchaseOrder{
		{Model: gorm.Model{ID: 1}, UserID: "demo-user", Status: constants.OrderStatusInTransit, ExpectedArrival: now.AddDate(0, 0, -3)},
		{Model: gorm.Model{ID: 2}, UserID: "demo-user", Status: constants.OrderStatusPartial, ExpectedArrival: now.AddDate(0, 0, 2)},
		{Model: gorm.Model{ID: 3}, UserID: "demo-user", Status: constants.OrderStatusCompleted, ExpectedArrival: now.AddDate(0, 0, -5), Arrivals: []model.Arrival{{ArrivedAt: now.AddDate(0, 0, -2)}}},
		{Model: gorm.Model{ID: 4}, UserID: "demo-user", Status: constants.OrderStatusPending, ExpectedArrival: now.AddDate(0, 0, -1)},
	}}
	svc := newPurchaseOrderService(repo, offers, products)
	views, err := svc.List("demo-user")
	if err != nil || len(views) != 4 {
		t.Fatalf("unexpected list: %v %d", err, len(views))
	}
	want := map[uint]int{1: 3, 2: 0, 3: 3, 4: 0}
	for _, view := range views {
		if view.DelayDays != want[view.ID] {
			t.Fatalf("order %d delay: got %d want %d", view.ID, view.DelayDays, want[view.ID])
		}
	}
}

func TestPurchaseOrderCancelForbiddenIsSentinel(t *testing.T) {
	offers, products := orderFixtures()
	repo := &mockPurchaseOrderRepo{order: model.PurchaseOrder{UserID: "demo-user", Status: constants.OrderStatusPending}}
	svc := newPurchaseOrderService(repo, offers, products)
	_, err := svc.Cancel(1, "someone-else")
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
