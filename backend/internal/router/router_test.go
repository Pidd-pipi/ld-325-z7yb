package router

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type testOrderView struct {
	ID               uint    `json:"ID"`
	Status           string  `json:"Status"`
	SupplierName     string  `json:"SupplierName"`
	UnitPrice        float64 `json:"UnitPrice"`
	Freight          string  `json:"Freight"`
	Quantity         int     `json:"Quantity"`
	ReceivedQuantity int     `json:"ReceivedQuantity"`
	RejectReason     string  `json:"RejectReason"`
	DelayDays        int     `json:"DelayDays"`
	Arrivals         []struct {
		Quantity int `json:"Quantity"`
	} `json:"Arrivals"`
}

func setupTestServer(t *testing.T) (*httptest.Server, model.Offer) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = model.Migrate(db); err != nil {
		t.Fatal(err)
	}
	category := model.Category{Name: "瓷砖"}
	supplier := model.Supplier{Name: "筑家优选旗舰店", Status: constants.SupplierApproved}
	product := model.Product{Name: "云纹岩板 900×1800", Unit: "片"}
	if err = db.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&supplier).Error; err != nil {
		t.Fatal(err)
	}
	product.CategoryID = category.ID
	if err = db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	offer := model.Offer{ProductID: product.ID, SupplierID: supplier.ID, UnitPrice: 398, MOQ: 10, Freight: "满 30 片送货上门", DeliveryDays: 3, StockStatus: constants.StatusInStock}
	if err = db.Create(&offer).Error; err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(db, slog.New(slog.NewTextHandler(io.Discard, nil)), "test-secret"))
	t.Cleanup(server.Close)
	return server, offer
}

func callAPI(t *testing.T, method, url, token string, body any) (int, testEnvelope) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var envelope testEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, envelope
}

func decodeData[T any](t *testing.T, envelope testEnvelope) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(envelope.Data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestPurchaseOrderFlow(t *testing.T) {
	server, offer := setupTestServer(t)
	base := server.URL + "/api/v1"
	create := func(quantity int) (int, testEnvelope) {
		return callAPI(t, http.MethodPost, base+"/purchase-orders", "", map[string]any{"offer_id": offer.ID, "quantity": quantity})
	}

	status, envelope := create(5)
	if status != http.StatusBadRequest || !strings.Contains(envelope.Message, "起订量") {
		t.Fatalf("below-MOQ order must be rejected: %d %s", status, envelope.Message)
	}

	status, envelope = create(20)
	if status != http.StatusOK {
		t.Fatalf("create failed: %d %s", status, envelope.Message)
	}
	order := decodeData[testOrderView](t, envelope)
	if order.Status != constants.OrderStatusPending || order.SupplierName != "筑家优选旗舰店" || order.UnitPrice != 398 || order.Freight != "满 30 片送货上门" {
		t.Fatalf("locked terms mismatch: %+v", order)
	}

	status, _ = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(order.ID)+"/arrivals", "", map[string]any{"quantity": 5})
	if status != http.StatusBadRequest {
		t.Fatalf("arrival before acceptance must fail: %d", status)
	}

	status, _ = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(order.ID)+"/accept", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("accept requires supplier role: %d", status)
	}

	status, envelope = callAPI(t, http.MethodPost, base+"/auth/demo-token", "", map[string]string{"role": "supplier"})
	if status != http.StatusOK {
		t.Fatalf("demo token failed: %d", status)
	}
	token := decodeData[struct {
		Token string `json:"token"`
	}](t, envelope).Token

	status, envelope = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(order.ID)+"/accept", token, nil)
	if status != http.StatusOK || decodeData[testOrderView](t, envelope).Status != constants.OrderStatusInTransit {
		t.Fatalf("accept failed: %d", status)
	}

	status, _ = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(order.ID)+"/cancel", "", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("cancel after acceptance must fail: %d", status)
	}

	status, envelope = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(order.ID)+"/arrivals", "", map[string]any{"quantity": 25})
	if status != http.StatusBadRequest || !strings.Contains(envelope.Message, "还可登记 20") {
		t.Fatalf("over-arrival must show remaining: %d %s", status, envelope.Message)
	}

	status, envelope = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(order.ID)+"/arrivals", "", map[string]any{"quantity": 8, "note": "第一批"})
	if status != http.StatusOK || decodeData[testOrderView](t, envelope).Status != constants.OrderStatusPartial {
		t.Fatalf("partial arrival failed: %d", status)
	}

	status, envelope = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(order.ID)+"/arrivals", "", map[string]any{"quantity": 12})
	order = decodeData[testOrderView](t, envelope)
	if status != http.StatusOK || order.Status != constants.OrderStatusCompleted || order.ReceivedQuantity != 20 || len(order.Arrivals) != 2 {
		t.Fatalf("completion failed: %d %+v", status, order)
	}

	status, envelope = callAPI(t, http.MethodGet, base+"/purchase-orders", "", nil)
	orders := decodeData[[]testOrderView](t, envelope)
	if status != http.StatusOK || len(orders) != 1 || orders[0].Status != constants.OrderStatusCompleted {
		t.Fatalf("list failed: %d %+v", status, orders)
	}
}

func TestPurchaseOrderCancelAndReject(t *testing.T) {
	server, offer := setupTestServer(t)
	base := server.URL + "/api/v1"

	status, envelope := callAPI(t, http.MethodPost, base+"/purchase-orders", "", map[string]any{"offer_id": offer.ID, "quantity": 15})
	if status != http.StatusOK {
		t.Fatalf("create failed: %d", status)
	}
	cancelled := decodeData[testOrderView](t, envelope)
	status, envelope = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(cancelled.ID)+"/cancel", "", nil)
	if status != http.StatusOK || decodeData[testOrderView](t, envelope).Status != constants.OrderStatusCancelled {
		t.Fatalf("cancel before acceptance must succeed: %d", status)
	}

	status, envelope = callAPI(t, http.MethodPost, base+"/purchase-orders", "", map[string]any{"offer_id": offer.ID, "quantity": 12})
	if status != http.StatusOK {
		t.Fatalf("create failed: %d", status)
	}
	rejected := decodeData[testOrderView](t, envelope)
	_, envelope = callAPI(t, http.MethodPost, base+"/auth/demo-token", "", map[string]string{"role": "supplier"})
	token := decodeData[struct {
		Token string `json:"token"`
	}](t, envelope).Token
	status, envelope = callAPI(t, http.MethodPost, base+"/purchase-orders/"+idString(rejected.ID)+"/reject", token, map[string]string{"reason": "岩板缺货，预计 20 天后恢复"})
	view := decodeData[testOrderView](t, envelope)
	if status != http.StatusOK || view.Status != constants.OrderStatusRejected || view.RejectReason != "岩板缺货，预计 20 天后恢复" {
		t.Fatalf("reject must keep reason: %d %+v", status, view)
	}

	status, envelope = callAPI(t, http.MethodGet, base+"/purchase-orders", "", nil)
	orders := decodeData[[]testOrderView](t, envelope)
	if status != http.StatusOK || len(orders) != 2 {
		t.Fatalf("list failed: %d %d", status, len(orders))
	}
	persisted := map[string]testOrderView{}
	for _, item := range orders {
		persisted[item.Status] = item
	}
	if persisted[constants.OrderStatusCancelled].ID != cancelled.ID {
		t.Fatalf("cancelled status not persisted: %+v", orders)
	}
	if persisted[constants.OrderStatusRejected].RejectReason != "岩板缺货，预计 20 天后恢复" {
		t.Fatalf("reject reason not persisted: %+v", orders)
	}
}

func idString(id uint) string {
	return strings.TrimSpace(jsonNumber(id))
}

func jsonNumber(id uint) string {
	payload, _ := json.Marshal(id)
	return string(payload)
}
