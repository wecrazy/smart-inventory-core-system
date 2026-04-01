package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	appconfig "github.com/wecrazy/smart-inventory-core-system/backend/internal/app/config"
	"github.com/wecrazy/smart-inventory-core-system/backend/internal/app/service"
	"github.com/wecrazy/smart-inventory-core-system/backend/internal/domain"
)

func TestHealthCheck(t *testing.T) {
	app := newTestApp(&stubInventoryStore{}, &stubTransactionStore{})

	response := performRequest(t, app, stdhttp.MethodGet, "/api/v1/health", "")
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}

	var payload struct {
		Data map[string]string `json:"data"`
	}
	decodeResponse(t, response, &payload)

	if payload.Data["status"] != "ok" {
		t.Fatalf("expected health status ok, got %q", payload.Data["status"])
	}
}

func TestSwaggerDocumentAvailable(t *testing.T) {
	app := newTestApp(&stubInventoryStore{}, &stubTransactionStore{})

	response := performRequest(t, app, stdhttp.MethodGet, "/swagger/doc.json", "")
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read swagger document: %v", err)
	}

	var document struct {
		BasePath string `json:"basePath"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("failed to decode swagger document: %v", err)
	}

	if document.BasePath != "/api/v1" {
		t.Fatalf("expected swagger document base path /api/v1, got %q", document.BasePath)
	}
	response.Body = io.NopCloser(bytes.NewBuffer(body))
}

func TestListInventoryNormalizesFilters(t *testing.T) {
	inventoryStore := &stubInventoryStore{}
	app := newTestApp(inventoryStore, &stubTransactionStore{})

	response := performRequest(t, app, stdhttp.MethodGet, "/api/v1/inventory?search=%20widget%20&sku=abc-123&customer=%20Acme%20", "")
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}

	if len(inventoryStore.listInventoryCalls) != 1 {
		t.Fatalf("expected 1 list inventory call, got %d", len(inventoryStore.listInventoryCalls))
	}

	filters := inventoryStore.listInventoryCalls[0]
	if filters.Search != "widget" {
		t.Fatalf("expected trimmed search filter, got %q", filters.Search)
	}
	if filters.SKU != "ABC-123" {
		t.Fatalf("expected uppercase sku filter, got %q", filters.SKU)
	}
	if filters.Customer != "Acme" {
		t.Fatalf("expected trimmed customer filter, got %q", filters.Customer)
	}
}

func TestCreateInventoryReturnsCreatedResource(t *testing.T) {
	now := time.Date(2026, time.April, 1, 10, 0, 0, 0, time.UTC)
	inventoryStore := &stubInventoryStore{
		createInventoryFunc: func(_ context.Context, input service.CreateInventoryInput) (domain.Inventory, error) {
			inventory := domain.Inventory{
				ID:            7,
				SKU:           input.SKU,
				Name:          input.Name,
				CustomerName:  input.CustomerName,
				PhysicalStock: input.PhysicalStock,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			inventory.RefreshAvailable()
			return inventory, nil
		},
	}

	app := newTestApp(inventoryStore, &stubTransactionStore{})
	payload := `{"sku":" sku-1 ","name":" Widget ","customerName":" ACME ","physicalStock":15}`

	response := performRequest(t, app, stdhttp.MethodPost, "/api/v1/inventory", payload)
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusCreated {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusCreated, response.StatusCode)
	}

	if len(inventoryStore.createInventoryCalls) != 1 {
		t.Fatalf("expected 1 create inventory call, got %d", len(inventoryStore.createInventoryCalls))
	}

	call := inventoryStore.createInventoryCalls[0]
	if call.SKU != "SKU-1" {
		t.Fatalf("expected normalized sku, got %q", call.SKU)
	}
	if call.Name != "Widget" {
		t.Fatalf("expected trimmed name, got %q", call.Name)
	}
	if call.CustomerName != "ACME" {
		t.Fatalf("expected trimmed customer name, got %q", call.CustomerName)
	}

	var body struct {
		Data domain.Inventory `json:"data"`
	}
	decodeResponse(t, response, &body)

	if body.Data.ID != 7 {
		t.Fatalf("expected inventory id 7, got %d", body.Data.ID)
	}
	if body.Data.AvailableStock != 15 {
		t.Fatalf("expected available stock 15, got %d", body.Data.AvailableStock)
	}
}

func TestCreateInventoryValidationError(t *testing.T) {
	inventoryStore := &stubInventoryStore{}
	app := newTestApp(inventoryStore, &stubTransactionStore{})
	payload := `{"sku":"sku-1","customerName":"ACME","physicalStock":5}`

	response := performRequest(t, app, stdhttp.MethodPost, "/api/v1/inventory", payload)
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusBadRequest, response.StatusCode)
	}
	if len(inventoryStore.createInventoryCalls) != 0 {
		t.Fatalf("expected create inventory store to not be called, got %d calls", len(inventoryStore.createInventoryCalls))
	}

	var body struct {
		Error string `json:"error"`
	}
	decodeResponse(t, response, &body)

	if !strings.Contains(body.Error, "name is required") {
		t.Fatalf("expected validation error for missing name, got %q", body.Error)
	}
}

func TestCancelStockInAcceptsEmptyBody(t *testing.T) {
	now := time.Date(2026, time.April, 1, 10, 0, 0, 0, time.UTC)
	transactionStore := &stubTransactionStore{
		cancelStockInFunc: func(_ context.Context, transactionID int64, note string) (domain.Transaction, error) {
			return domain.Transaction{
				ID:            transactionID,
				Type:          domain.TypeStockIn,
				Status:        domain.StatusCancelled,
				ReferenceCode: "IN-20260401-100000.000000",
				Note:          note,
				CreatedAt:     now,
				UpdatedAt:     now,
			}, nil
		},
	}

	app := newTestApp(&stubInventoryStore{}, transactionStore)

	response := performRequest(t, app, stdhttp.MethodPost, "/api/v1/stock-in/42/cancel", "")
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}
	if len(transactionStore.cancelStockInCalls) != 1 {
		t.Fatalf("expected 1 cancel stock-in call, got %d", len(transactionStore.cancelStockInCalls))
	}

	call := transactionStore.cancelStockInCalls[0]
	if call.transactionID != 42 {
		t.Fatalf("expected transaction id 42, got %d", call.transactionID)
	}
	if call.note != "" {
		t.Fatalf("expected empty cancellation note, got %q", call.note)
	}

	var body struct {
		Data domain.Transaction `json:"data"`
	}
	decodeResponse(t, response, &body)

	if body.Data.Status != domain.StatusCancelled {
		t.Fatalf("expected cancelled status, got %s", body.Data.Status)
	}
	if body.Data.Note != "" {
		t.Fatalf("expected empty note in response, got %q", body.Data.Note)
	}
}

func TestUpdateStockOutStatusPromotesTransaction(t *testing.T) {
	now := time.Date(2026, time.April, 1, 11, 0, 0, 0, time.UTC)
	transactionStore := &stubTransactionStore{
		updateStockOutStatusFunc: func(_ context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error) {
			return domain.Transaction{
				ID:            transactionID,
				Type:          domain.TypeStockOut,
				Status:        next,
				ReferenceCode: "OUT-20260401-110000.000000",
				Note:          note,
				CreatedAt:     now,
				UpdatedAt:     now,
			}, nil
		},
	}

	app := newTestApp(&stubInventoryStore{}, transactionStore)
	payload := `{"status":"IN_PROGRESS","note":"picked for packing"}`

	response := performRequest(t, app, stdhttp.MethodPatch, "/api/v1/stock-out/9/status", payload)
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}
	if len(transactionStore.updateStockOutStatusCalls) != 1 {
		t.Fatalf("expected 1 stock-out status update call, got %d", len(transactionStore.updateStockOutStatusCalls))
	}

	call := transactionStore.updateStockOutStatusCalls[0]
	if call.transactionID != 9 {
		t.Fatalf("expected transaction id 9, got %d", call.transactionID)
	}
	if call.status != domain.StatusInProgress {
		t.Fatalf("expected status %s, got %s", domain.StatusInProgress, call.status)
	}
	if call.note != "picked for packing" {
		t.Fatalf("expected note to be forwarded, got %q", call.note)
	}

	var body struct {
		Data domain.Transaction `json:"data"`
	}
	decodeResponse(t, response, &body)

	if body.Data.Status != domain.StatusInProgress {
		t.Fatalf("expected response status %s, got %s", domain.StatusInProgress, body.Data.Status)
	}
	if body.Data.Note != "picked for packing" {
		t.Fatalf("expected response note to be forwarded, got %q", body.Data.Note)
	}
}

func TestUpdateStockOutStatusConflictMapsToConflictResponse(t *testing.T) {
	transactionStore := &stubTransactionStore{
		updateStockOutStatusFunc: func(_ context.Context, _ int64, _ domain.TransactionStatus, _ string) (domain.Transaction, error) {
			return domain.Transaction{}, service.Conflict("invalid status transition")
		},
	}

	app := newTestApp(&stubInventoryStore{}, transactionStore)
	payload := `{"status":"DONE"}`

	response := performRequest(t, app, stdhttp.MethodPatch, "/api/v1/stock-out/3/status", payload)
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusConflict {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusConflict, response.StatusCode)
	}

	var body struct {
		Error string `json:"error"`
	}
	decodeResponse(t, response, &body)

	if !strings.Contains(body.Error, "invalid status transition") {
		t.Fatalf("expected conflict error body, got %q", body.Error)
	}
}

func TestCancelStockOutAcceptsEmptyBody(t *testing.T) {
	now := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	transactionStore := &stubTransactionStore{
		cancelStockOutFunc: func(_ context.Context, transactionID int64, note string) (domain.Transaction, error) {
			return domain.Transaction{
				ID:            transactionID,
				Type:          domain.TypeStockOut,
				Status:        domain.StatusCancelled,
				ReferenceCode: "OUT-20260401-120000.000000",
				Note:          note,
				CreatedAt:     now,
				UpdatedAt:     now,
			}, nil
		},
	}

	app := newTestApp(&stubInventoryStore{}, transactionStore)

	response := performRequest(t, app, stdhttp.MethodPost, "/api/v1/stock-out/13/cancel", "")
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}
	if len(transactionStore.cancelStockOutCalls) != 1 {
		t.Fatalf("expected 1 stock-out cancel call, got %d", len(transactionStore.cancelStockOutCalls))
	}

	call := transactionStore.cancelStockOutCalls[0]
	if call.transactionID != 13 {
		t.Fatalf("expected transaction id 13, got %d", call.transactionID)
	}
	if call.note != "" {
		t.Fatalf("expected empty cancellation note, got %q", call.note)
	}

	var body struct {
		Data domain.Transaction `json:"data"`
	}
	decodeResponse(t, response, &body)

	if body.Data.Status != domain.StatusCancelled {
		t.Fatalf("expected cancelled status, got %s", body.Data.Status)
	}
}

func TestUpdateStockOutStatusRejectsInvalidID(t *testing.T) {
	transactionStore := &stubTransactionStore{}
	app := newTestApp(&stubInventoryStore{}, transactionStore)
	payload := `{"status":"IN_PROGRESS"}`

	response := performRequest(t, app, stdhttp.MethodPatch, "/api/v1/stock-out/not-a-number/status", payload)
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusBadRequest, response.StatusCode)
	}
	if len(transactionStore.updateStockOutStatusCalls) != 0 {
		t.Fatalf("expected no stock-out status update calls, got %d", len(transactionStore.updateStockOutStatusCalls))
	}

	var body struct {
		Error string `json:"error"`
	}
	decodeResponse(t, response, &body)

	if !strings.Contains(body.Error, "id must be a positive integer") {
		t.Fatalf("expected invalid id error, got %q", body.Error)
	}
}

func TestListReportsAcceptsPagination(t *testing.T) {
	now := time.Date(2026, time.April, 1, 13, 0, 0, 0, time.UTC)
	transactionStore := &stubTransactionStore{
		listReportsFunc: func(_ context.Context, filters service.ReportFilters) (service.ReportPage, error) {
			return service.ReportPage{
				Items: []domain.Transaction{{
					ID:            17,
					Type:          domain.TypeStockOut,
					Status:        domain.StatusDone,
					ReferenceCode: "OUT-20260401-130000.000000",
					Note:          "completed shipment",
					CreatedAt:     now,
					UpdatedAt:     now,
					CompletedAt:   &now,
				}},
				Total:    21,
				Limit:    filters.Limit,
				Offset:   filters.Offset,
				UnitsIn:  12,
				UnitsOut: 9,
			}, nil
		},
	}

	app := newTestApp(&stubInventoryStore{}, transactionStore)

	response := performRequest(t, app, stdhttp.MethodGet, "/api/v1/reports?limit=5&offset=10", "")
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}
	if len(transactionStore.listReportsCalls) != 1 {
		t.Fatalf("expected 1 list reports call, got %d", len(transactionStore.listReportsCalls))
	}

	call := transactionStore.listReportsCalls[0]
	if call.limit != 5 {
		t.Fatalf("expected limit 5, got %d", call.limit)
	}
	if call.offset != 10 {
		t.Fatalf("expected offset 10, got %d", call.offset)
	}

	var body struct {
		Data struct {
			Items    []domain.Transaction `json:"items"`
			Total    int64                `json:"total"`
			Limit    int                  `json:"limit"`
			Offset   int                  `json:"offset"`
			UnitsIn  int64                `json:"unitsIn"`
			UnitsOut int64                `json:"unitsOut"`
		} `json:"data"`
	}
	decodeResponse(t, response, &body)

	if body.Data.Total != 21 {
		t.Fatalf("expected total 21, got %d", body.Data.Total)
	}
	if body.Data.Limit != 5 {
		t.Fatalf("expected response limit 5, got %d", body.Data.Limit)
	}
	if body.Data.Offset != 10 {
		t.Fatalf("expected response offset 10, got %d", body.Data.Offset)
	}
	if body.Data.UnitsIn != 12 {
		t.Fatalf("expected unitsIn 12, got %d", body.Data.UnitsIn)
	}
	if body.Data.UnitsOut != 9 {
		t.Fatalf("expected unitsOut 9, got %d", body.Data.UnitsOut)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].ID != 17 {
		t.Fatalf("expected paged report item with id 17, got %+v", body.Data.Items)
	}
}

func TestListReportsAcceptsFilters(t *testing.T) {
	transactionStore := &stubTransactionStore{
		listReportsFunc: func(_ context.Context, filters service.ReportFilters) (service.ReportPage, error) {
			return service.ReportPage{Items: []domain.Transaction{}, Total: 0, Limit: filters.Limit, Offset: filters.Offset}, nil
		},
	}

	app := newTestApp(&stubInventoryStore{}, transactionStore)

	response := performRequest(t, app, stdhttp.MethodGet, "/api/v1/reports?type=stock_out&referenceCode=out-manual&completedFrom=2026-04-01&completedTo=2026-04-02", "")
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}
	if len(transactionStore.listReportsCalls) != 1 {
		t.Fatalf("expected 1 list reports call, got %d", len(transactionStore.listReportsCalls))
	}

	call := transactionStore.listReportsCalls[0]
	if call.typeValue != string(domain.TypeStockOut) {
		t.Fatalf("expected type filter %s, got %s", domain.TypeStockOut, call.typeValue)
	}
	if call.referenceCode != "OUT-MANUAL" {
		t.Fatalf("expected uppercase reference code filter, got %q", call.referenceCode)
	}
	if call.completedFrom != "2026-04-01T00:00:00Z" {
		t.Fatalf("expected completedFrom at start of day, got %q", call.completedFrom)
	}
	if call.completedTo != "2026-04-03T00:00:00Z" {
		t.Fatalf("expected completedTo to be exclusive next day, got %q", call.completedTo)
	}
}

func TestExportReportsReturnsCSVAttachment(t *testing.T) {
	now := time.Date(2026, time.April, 1, 13, 0, 0, 0, time.UTC)
	transactionStore := &stubTransactionStore{
		listAllReportsFunc: func(_ context.Context, filters service.ReportFilters) ([]domain.Transaction, error) {
			if filters.ReferenceCode != "OUT-MANUAL" {
				t.Fatalf("expected uppercase export reference code filter, got %q", filters.ReferenceCode)
			}

			return []domain.Transaction{{
				ID:            17,
				Type:          domain.TypeStockOut,
				Status:        domain.StatusDone,
				ReferenceCode: "OUT-MANUAL-002",
				Note:          "final shipment",
				CreatedAt:     now,
				UpdatedAt:     now,
				CompletedAt:   &now,
				Items: []domain.TransactionItem{{
					InventoryID:  1,
					SKU:          "SKU-001",
					Name:         "Widget A",
					CustomerName: "Acme Corp",
					Quantity:     20,
				}},
				History: []domain.HistoryEntry{{
					Status:    domain.StatusDone,
					Note:      "delivered",
					CreatedAt: now,
				}},
			}}, nil
		},
	}

	app := newTestApp(&stubInventoryStore{}, transactionStore)

	response := performRequest(t, app, stdhttp.MethodGet, "/api/v1/reports/export?referenceCode=out-manual", "")
	defer response.Body.Close()

	if response.StatusCode != stdhttp.StatusOK {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusOK, response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.Contains(contentType, "text/csv") {
		t.Fatalf("expected text/csv content type, got %q", contentType)
	}
	if contentDisposition := response.Header.Get("Content-Disposition"); !strings.Contains(contentDisposition, "attachment;") {
		t.Fatalf("expected attachment content disposition, got %q", contentDisposition)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read csv response body: %v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "transaction_id,reference_code,transaction_type") {
		t.Fatalf("expected csv header row, got %q", body)
	}
	if !strings.Contains(body, "OUT-MANUAL-002") {
		t.Fatalf("expected exported reference code, got %q", body)
	}
	if !strings.Contains(body, "SKU-001") {
		t.Fatalf("expected exported SKU, got %q", body)
	}
}

func newTestApp(inventoryStore service.InventoryStore, transactionStore service.TransactionStore) *fiber.App {
	testService := service.New(inventoryStore, transactionStore)
	return NewApp(appconfig.Config{Environment: "test"}, testService)
}

func performRequest(t *testing.T, app *fiber.App, method string, target string, body string) *stdhttp.Response {
	t.Helper()

	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := app.Test(request, fiber.TestConfig{Timeout: 0})
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	return response
}

func decodeResponse(t *testing.T, response *stdhttp.Response, target any) {
	t.Helper()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("failed to decode response body %s: %v", string(body), err)
	}

	response.Body = io.NopCloser(bytes.NewBuffer(body))
}

type stubInventoryStore struct {
	listInventoryFunc   func(ctx context.Context, filters service.InventoryFilters) ([]domain.Inventory, error)
	createInventoryFunc func(ctx context.Context, input service.CreateInventoryInput) (domain.Inventory, error)
	adjustInventoryFunc func(ctx context.Context, input service.AdjustInventoryInput) (domain.Transaction, error)

	listInventoryCalls   []service.InventoryFilters
	createInventoryCalls []service.CreateInventoryInput
	adjustInventoryCalls []service.AdjustInventoryInput
}

func (store *stubInventoryStore) ListInventory(ctx context.Context, filters service.InventoryFilters) ([]domain.Inventory, error) {
	store.listInventoryCalls = append(store.listInventoryCalls, filters)
	if store.listInventoryFunc != nil {
		return store.listInventoryFunc(ctx, filters)
	}

	return nil, nil
}

func (store *stubInventoryStore) CreateInventory(ctx context.Context, input service.CreateInventoryInput) (domain.Inventory, error) {
	store.createInventoryCalls = append(store.createInventoryCalls, input)
	if store.createInventoryFunc != nil {
		return store.createInventoryFunc(ctx, input)
	}

	return domain.Inventory{}, nil
}

func (store *stubInventoryStore) AdjustInventory(ctx context.Context, input service.AdjustInventoryInput) (domain.Transaction, error) {
	store.adjustInventoryCalls = append(store.adjustInventoryCalls, input)
	if store.adjustInventoryFunc != nil {
		return store.adjustInventoryFunc(ctx, input)
	}

	return domain.Transaction{}, nil
}

type cancelTransactionCall struct {
	transactionID int64
	note          string
}

type updateStatusCall struct {
	transactionID int64
	status        domain.TransactionStatus
	note          string
}

type reportFiltersCall struct {
	limit         int
	offset        int
	referenceCode string
	typeValue     string
	completedFrom string
	completedTo   string
}

type stubTransactionStore struct {
	createStockInFunc        func(ctx context.Context, input service.CreateTransactionInput) (domain.Transaction, error)
	updateStockInStatusFunc  func(ctx context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error)
	cancelStockInFunc        func(ctx context.Context, transactionID int64, note string) (domain.Transaction, error)
	createStockOutFunc       func(ctx context.Context, input service.CreateTransactionInput) (domain.Transaction, error)
	updateStockOutStatusFunc func(ctx context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error)
	cancelStockOutFunc       func(ctx context.Context, transactionID int64, note string) (domain.Transaction, error)
	getTransactionFunc       func(ctx context.Context, transactionID int64) (domain.Transaction, error)
	listTransactionsFunc     func(ctx context.Context, transactionType domain.TransactionType, status *domain.TransactionStatus) ([]domain.Transaction, error)
	listReportsFunc          func(ctx context.Context, filters service.ReportFilters) (service.ReportPage, error)
	listAllReportsFunc       func(ctx context.Context, filters service.ReportFilters) ([]domain.Transaction, error)

	updateStockOutStatusCalls []updateStatusCall
	cancelStockInCalls        []cancelTransactionCall
	cancelStockOutCalls       []cancelTransactionCall
	listReportsCalls          []reportFiltersCall
	listAllReportsCalls       []reportFiltersCall
}

func (store *stubTransactionStore) CreateStockIn(ctx context.Context, input service.CreateTransactionInput) (domain.Transaction, error) {
	if store.createStockInFunc != nil {
		return store.createStockInFunc(ctx, input)
	}

	return domain.Transaction{}, nil
}

func (store *stubTransactionStore) UpdateStockInStatus(ctx context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error) {
	if store.updateStockInStatusFunc != nil {
		return store.updateStockInStatusFunc(ctx, transactionID, next, note)
	}

	return domain.Transaction{}, nil
}

func (store *stubTransactionStore) CancelStockIn(ctx context.Context, transactionID int64, note string) (domain.Transaction, error) {
	store.cancelStockInCalls = append(store.cancelStockInCalls, cancelTransactionCall{transactionID: transactionID, note: note})
	if store.cancelStockInFunc != nil {
		return store.cancelStockInFunc(ctx, transactionID, note)
	}

	return domain.Transaction{}, nil
}

func (store *stubTransactionStore) CreateStockOut(ctx context.Context, input service.CreateTransactionInput) (domain.Transaction, error) {
	if store.createStockOutFunc != nil {
		return store.createStockOutFunc(ctx, input)
	}

	return domain.Transaction{}, nil
}

func (store *stubTransactionStore) UpdateStockOutStatus(ctx context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error) {
	store.updateStockOutStatusCalls = append(store.updateStockOutStatusCalls, updateStatusCall{transactionID: transactionID, status: next, note: note})
	if store.updateStockOutStatusFunc != nil {
		return store.updateStockOutStatusFunc(ctx, transactionID, next, note)
	}

	return domain.Transaction{}, nil
}

func (store *stubTransactionStore) CancelStockOut(ctx context.Context, transactionID int64, note string) (domain.Transaction, error) {
	store.cancelStockOutCalls = append(store.cancelStockOutCalls, cancelTransactionCall{transactionID: transactionID, note: note})
	if store.cancelStockOutFunc != nil {
		return store.cancelStockOutFunc(ctx, transactionID, note)
	}

	return domain.Transaction{}, nil
}

func (store *stubTransactionStore) GetTransaction(ctx context.Context, transactionID int64) (domain.Transaction, error) {
	if store.getTransactionFunc != nil {
		return store.getTransactionFunc(ctx, transactionID)
	}

	return domain.Transaction{}, nil
}

func (store *stubTransactionStore) ListTransactions(ctx context.Context, transactionType domain.TransactionType, status *domain.TransactionStatus) ([]domain.Transaction, error) {
	if store.listTransactionsFunc != nil {
		return store.listTransactionsFunc(ctx, transactionType, status)
	}

	return nil, nil
}

func (store *stubTransactionStore) ListReports(ctx context.Context, filters service.ReportFilters) (service.ReportPage, error) {
	store.listReportsCalls = append(store.listReportsCalls, reportFiltersCall{
		limit:         filters.Limit,
		offset:        filters.Offset,
		referenceCode: filters.ReferenceCode,
		typeValue:     reportTypeString(filters.Type),
		completedFrom: reportTimeString(filters.CompletedFrom),
		completedTo:   reportTimeString(filters.CompletedTo),
	})
	if store.listReportsFunc != nil {
		return store.listReportsFunc(ctx, filters)
	}

	return service.ReportPage{}, nil
}

func (store *stubTransactionStore) ListAllReports(ctx context.Context, filters service.ReportFilters) ([]domain.Transaction, error) {
	store.listAllReportsCalls = append(store.listAllReportsCalls, reportFiltersCall{
		limit:         filters.Limit,
		offset:        filters.Offset,
		referenceCode: filters.ReferenceCode,
		typeValue:     reportTypeString(filters.Type),
		completedFrom: reportTimeString(filters.CompletedFrom),
		completedTo:   reportTimeString(filters.CompletedTo),
	})
	if store.listAllReportsFunc != nil {
		return store.listAllReportsFunc(ctx, filters)
	}

	return nil, nil
}

func reportTypeString(reportType *domain.TransactionType) string {
	if reportType == nil {
		return ""
	}

	return string(*reportType)
}

func reportTimeString(value *time.Time) string {
	if value == nil {
		return ""
	}

	return value.UTC().Format(time.RFC3339)
}
