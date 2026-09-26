package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOrderRepo(t *testing.T) PurchaseOrderRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = model.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return NewPurchaseOrderRepository(db)
}

func TestPurchaseOrderRepositoryDeliveryCap(t *testing.T) {
	repo := setupOrderRepo(t)
	order, err := repo.Create(model.PurchaseOrder{
		OrderNo: "PO20260926-T001", UserID: "demo-user", OfferID: 1, ProductID: 1, SupplierID: 1,
		Quantity: 10, UnitPrice: 88, Freight: "包邮", ExpectedArrival: time.Now().AddDate(0, 0, 2),
		Status: constants.OrderStatusInTransit,
	})
	if err != nil {
		t.Fatal(err)
	}
	order.ReceivedTotal = 6
	order.Status = constants.OrderStatusPartial
	if _, err = repo.RegisterDelivery(order, model.Delivery{PurchaseOrderID: order.ID, Quantity: 6}); err != nil {
		t.Fatal(err)
	}
	order.ReceivedTotal = 10
	order.Status = constants.OrderStatusCompleted
	if _, err = repo.RegisterDelivery(order, model.Delivery{PurchaseOrderID: order.ID, Quantity: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RegisterDelivery(order, model.Delivery{PurchaseOrderID: order.ID, Quantity: 1}); !errors.Is(err, apperrors.ErrExceedsOrdered) {
		t.Fatalf("expected exceeds error, got %v", err)
	}
	reloaded, err := repo.GetByID(order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ReceivedTotal != 10 || len(reloaded.Deliveries) != 2 {
		t.Fatalf("unexpected state: total=%d deliveries=%d", reloaded.ReceivedTotal, len(reloaded.Deliveries))
	}
}

func TestPurchaseOrderRepositoryListFilter(t *testing.T) {
	repo := setupOrderRepo(t)
	base := model.PurchaseOrder{UserID: "demo-user", Quantity: 5, UnitPrice: 10, ExpectedArrival: time.Now()}
	base.OrderNo = "PO20260926-T101"
	base.Status = constants.OrderStatusPending
	if _, err := repo.Create(base); err != nil {
		t.Fatal(err)
	}
	base.OrderNo = "PO20260926-T102"
	base.Status = constants.OrderStatusCompleted
	if _, err := repo.Create(base); err != nil {
		t.Fatal(err)
	}
	all, err := repo.ListByUser("demo-user", "")
	if err != nil || len(all) != 2 {
		t.Fatalf("expected 2 orders, got %d %v", len(all), err)
	}
	pending, err := repo.ListByUser("demo-user", constants.OrderStatusPending)
	if err != nil || len(pending) != 1 || pending[0].Status != constants.OrderStatusPending {
		t.Fatalf("status filter failed: %+v %v", pending, err)
	}
	if _, err = repo.GetByID(9999); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
