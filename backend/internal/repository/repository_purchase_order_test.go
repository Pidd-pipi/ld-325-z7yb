package repository

import (
	"testing"
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPurchaseOrderDB(t *testing.T) (*gorm.DB, model.Product) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = model.Migrate(db); err != nil {
		t.Fatal(err)
	}
	product := model.Product{Name: "云纹岩板 900×1800", Unit: "片"}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	return db, product
}

func TestPurchaseOrderRepositoryCreateAndList(t *testing.T) {
	db, product := setupPurchaseOrderDB(t)
	repo := NewPurchaseOrderRepository(db)
	order := model.PurchaseOrder{UserID: "demo-user", ProductID: product.ID, SupplierName: "筑家优选旗舰店", UnitPrice: 398, Freight: "满 30 片送货上门", Quantity: 30, Status: constants.OrderStatusPending, ExpectedArrival: time.Now().AddDate(0, 0, 3)}
	created, err := repo.Create(order)
	if err != nil || created.ID == 0 {
		t.Fatalf("create failed: %v", err)
	}
	rows, err := repo.ListByUser("demo-user")
	if err != nil || len(rows) != 1 {
		t.Fatalf("list failed: %v %d", err, len(rows))
	}
	if rows[0].Product.Name != product.Name {
		t.Fatalf("product not preloaded: %+v", rows[0])
	}
	others, err := repo.ListByUser("someone-else")
	if err != nil || len(others) != 0 {
		t.Fatalf("orders leaked across users: %v %d", err, len(others))
	}
}

func TestPurchaseOrderRepositoryArrivalFlow(t *testing.T) {
	db, product := setupPurchaseOrderDB(t)
	repo := NewPurchaseOrderRepository(db)
	order, err := repo.Create(model.PurchaseOrder{UserID: "demo-user", ProductID: product.ID, Quantity: 10, Status: constants.OrderStatusInTransit, ExpectedArrival: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := repo.RegisterArrival(order.ID, model.Arrival{Quantity: 4, Note: "第一批", ArrivedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ReceivedQuantity != 4 || updated.Status != constants.OrderStatusPartial {
		t.Fatalf("expected partial with 4 received: %+v", updated)
	}
	if len(updated.Arrivals) != 1 || updated.Arrivals[0].Quantity != 4 {
		t.Fatalf("arrival not recorded: %+v", updated.Arrivals)
	}
	updated, err = repo.RegisterArrival(order.ID, model.Arrival{Quantity: 6, ArrivedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != constants.OrderStatusCompleted || updated.ReceivedQuantity != 10 {
		t.Fatalf("expected completed with 10 received: %+v", updated)
	}
	if _, err = repo.RegisterArrival(order.ID, model.Arrival{Quantity: 1, ArrivedAt: time.Now()}); err != apperrors.ErrInvalidTransition {
		t.Fatalf("completed order must reject arrivals: %v", err)
	}
}

func TestPurchaseOrderRepositoryArrivalExceeds(t *testing.T) {
	db, product := setupPurchaseOrderDB(t)
	repo := NewPurchaseOrderRepository(db)
	order, err := repo.Create(model.PurchaseOrder{UserID: "demo-user", ProductID: product.ID, Quantity: 10, Status: constants.OrderStatusInTransit, ExpectedArrival: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RegisterArrival(order.ID, model.Arrival{Quantity: 11, ArrivedAt: time.Now()}); err != apperrors.ErrArrivalExceeds {
		t.Fatalf("expected ErrArrivalExceeds, got %v", err)
	}
	reloaded, err := repo.Get(order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ReceivedQuantity != 0 || reloaded.Status != constants.OrderStatusInTransit || len(reloaded.Arrivals) != 0 {
		t.Fatalf("failed arrival must not persist: %+v", reloaded)
	}
}

func TestPurchaseOrderRepositoryArrivalRequiresAcceptance(t *testing.T) {
	db, product := setupPurchaseOrderDB(t)
	repo := NewPurchaseOrderRepository(db)
	order, err := repo.Create(model.PurchaseOrder{UserID: "demo-user", ProductID: product.ID, Quantity: 10, Status: constants.OrderStatusPending, ExpectedArrival: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RegisterArrival(order.ID, model.Arrival{Quantity: 2, ArrivedAt: time.Now()}); err != apperrors.ErrInvalidTransition {
		t.Fatalf("pending order must reject arrivals: %v", err)
	}
}
