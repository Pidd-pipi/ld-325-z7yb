package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = model.Migrate(db); err != nil {
		t.Fatal(err)
	}
	supplier := model.Supplier{Name: "筑家优选旗舰店", Status: constants.SupplierApproved}
	if err = db.Create(&supplier).Error; err != nil {
		t.Fatal(err)
	}
	product := model.Product{Name: "橡木多层地板 15mm", Unit: "㎡"}
	if err = db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	offer := model.Offer{ProductID: product.ID, SupplierID: supplier.ID, UnitPrice: 188, MOQ: 20, Freight: "满 100 ㎡免运费", DeliveryDays: 5, StockStatus: constants.StatusInStock}
	if err = db.Create(&offer).Error; err != nil {
		t.Fatal(err)
	}
	app := New(db, slog.New(slog.NewTextHandler(io.Discard, nil)), "test-secret")
	return httptest.NewServer(app)
}

func callAPI(t *testing.T, server *httptest.Server, method, path string, body any) (int, dto.Response) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var envelope dto.Response
	if err = json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, envelope
}

func TestPurchaseOrderLifecycle(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	status, envelope := callAPI(t, server, http.MethodPost, "/api/v1/orders", map[string]any{"offer_id": 1, "quantity": 5})
	if status != http.StatusBadRequest || envelope.Code != constants.ErrorBusiness {
		t.Fatalf("below MOQ must fail with business error: %d %+v", status, envelope)
	}

	status, envelope = callAPI(t, server, http.MethodPost, "/api/v1/orders", map[string]any{"offer_id": 1, "quantity": 40})
	if status != http.StatusOK {
		t.Fatalf("create failed: %d %+v", status, envelope)
	}
	var created dto.PurchaseOrderView
	if err := remarshal(envelope.Data, &created); err != nil {
		t.Fatal(err)
	}
	if created.Status != constants.OrderStatusPending || created.UnitPrice != 188 || created.Freight != "满 100 ㎡免运费" || created.Supplier.Name != "筑家优选旗舰店" {
		t.Fatalf("order terms not locked: %+v", created)
	}

	status, _ = callAPI(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/deliveries", created.ID), map[string]any{"quantity": 5})
	if status != http.StatusBadRequest {
		t.Fatal("pending order must not accept deliveries")
	}

	status, _ = callAPI(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/accept", created.ID), nil)
	if status != http.StatusOK {
		t.Fatal("accept failed")
	}
	status, _ = callAPI(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/cancel", created.ID), nil)
	if status != http.StatusBadRequest {
		t.Fatal("cancel after accept must fail")
	}

	status, envelope = callAPI(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/deliveries", created.ID), map[string]any{"quantity": 15, "note": "首批"})
	if status != http.StatusOK {
		t.Fatalf("first delivery failed: %d %+v", status, envelope)
	}
	var partial dto.PurchaseOrderView
	if err := remarshal(envelope.Data, &partial); err != nil {
		t.Fatal(err)
	}
	if partial.Status != constants.OrderStatusPartial || partial.ReceivedTotal != 15 {
		t.Fatalf("expected partial 15, got %+v", partial)
	}

	status, _ = callAPI(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/deliveries", created.ID), map[string]any{"quantity": 26})
	if status != http.StatusBadRequest {
		t.Fatal("cumulative deliveries must not exceed ordered quantity")
	}

	status, envelope = callAPI(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/deliveries", created.ID), map[string]any{"quantity": 25})
	if status != http.StatusOK {
		t.Fatalf("final delivery failed: %d %+v", status, envelope)
	}
	var done dto.PurchaseOrderView
	if err := remarshal(envelope.Data, &done); err != nil {
		t.Fatal(err)
	}
	if done.Status != constants.OrderStatusCompleted || done.ReceivedTotal != 40 || len(done.Deliveries) != 2 {
		t.Fatalf("expected completed order, got %+v", done)
	}

	status, envelope = callAPI(t, server, http.MethodGet, "/api/v1/orders?status=completed", nil)
	if status != http.StatusOK {
		t.Fatal("list by status failed")
	}
	var completed []dto.PurchaseOrderView
	if err := remarshal(envelope.Data, &completed); err != nil {
		t.Fatal(err)
	}
	if len(completed) != 1 {
		t.Fatalf("expected 1 completed order, got %d", len(completed))
	}
}

func TestPurchaseOrderRejectAndCancel(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	_, envelope := callAPI(t, server, http.MethodPost, "/api/v1/orders", map[string]any{"offer_id": 1, "quantity": 20})
	var first dto.PurchaseOrderView
	if err := remarshal(envelope.Data, &first); err != nil {
		t.Fatal(err)
	}
	status, envelope := callAPI(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/reject", first.ID), map[string]any{"reason": "厂家排产已满"})
	if status != http.StatusOK {
		t.Fatalf("reject failed: %d %+v", status, envelope)
	}
	var rejected dto.PurchaseOrderView
	if err := remarshal(envelope.Data, &rejected); err != nil {
		t.Fatal(err)
	}
	if rejected.Status != constants.OrderStatusRejected || rejected.RejectReason != "厂家排产已满" {
		t.Fatalf("reject reason not kept: %+v", rejected)
	}

	_, envelope = callAPI(t, server, http.MethodPost, "/api/v1/orders", map[string]any{"offer_id": 1, "quantity": 30})
	var second dto.PurchaseOrderView
	if err := remarshal(envelope.Data, &second); err != nil {
		t.Fatal(err)
	}
	status, envelope = callAPI(t, server, http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/cancel", second.ID), nil)
	if status != http.StatusOK {
		t.Fatalf("cancel before accept failed: %d %+v", status, envelope)
	}
	var cancelled dto.PurchaseOrderView
	if err := remarshal(envelope.Data, &cancelled); err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != constants.OrderStatusCancelled {
		t.Fatalf("expected cancelled, got %+v", cancelled)
	}

	status, _ = callAPI(t, server, http.MethodGet, "/api/v1/orders?status=bogus", nil)
	if status != http.StatusBadRequest {
		t.Fatal("unknown status filter must fail validation")
	}
}

func remarshal(data any, target any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}
