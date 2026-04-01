package http

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/wecrazy/smart-inventory-core-system/backend/internal/app/service"
	"github.com/wecrazy/smart-inventory-core-system/backend/internal/domain"
)

type handler struct {
	service *service.Service
}

type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type createInventoryRequest struct {
	SKU           string `json:"sku" example:"SKU-001"`
	Name          string `json:"name" example:"Widget A"`
	CustomerName  string `json:"customerName" example:"Acme Corp"`
	PhysicalStock int64  `json:"physicalStock" example:"120"`
}

type adjustInventoryRequest struct {
	InventoryID      int64  `json:"inventoryId" example:"1"`
	NewPhysicalStock int64  `json:"newPhysicalStock" example:"140"`
	ReferenceCode    string `json:"referenceCode" example:"ADJ-20260401-001"`
	Note             string `json:"note" example:"Cycle count correction"`
}

type transactionItemRequest struct {
	InventoryID int64 `json:"inventoryId" example:"1"`
	Quantity    int64 `json:"quantity" example:"10"`
}

type createTransactionRequest struct {
	ReferenceCode string                   `json:"referenceCode" example:"IN-20260401-001"`
	Note          string                   `json:"note" example:"Morning warehouse receipt"`
	Items         []transactionItemRequest `json:"items"`
}

type updateStatusRequest struct {
	Status string `json:"status" example:"IN_PROGRESS" enums:"IN_PROGRESS,DONE"`
	Note   string `json:"note" example:"Checked and ready for next step"`
}

type cancelTransactionRequest struct {
	Note string `json:"note" example:"Cancelled by operator"`
}

type reportListResponse struct {
	Items    []domain.Transaction `json:"items"`
	Total    int64                `json:"total" example:"25"`
	Limit    int                  `json:"limit" example:"10"`
	Offset   int                  `json:"offset" example:"0"`
	UnitsIn  int64                `json:"unitsIn" example:"220"`
	UnitsOut int64                `json:"unitsOut" example:"180"`
}

type healthStatus struct {
	Status string `json:"status" example:"ok"`
}

func newHandler(service *service.Service) *handler {
	return &handler{service: service}
}

// healthCheck returns the service status.
//
// @Summary      Health check
// @Description  Returns a lightweight status response for local smoke checks and uptime monitoring.
// @Tags         System
// @Produce      json
// @Success      200  {object}  envelope{data=healthStatus}
// @Router       /health [get]
func (handler *handler) healthCheck(ctx fiber.Ctx) error {
	return ctx.JSON(envelope{Data: healthStatus{Status: "ok"}})
}

// listInventory returns inventory items with optional filters.
//
// @Summary      List inventory
// @Description  Returns inventory items filtered by free-text search, SKU, and customer.
// @Tags         Inventory
// @Produce      json
// @Param        search    query     string  false  "Search by inventory name or SKU"
// @Param        sku       query     string  false  "Filter by SKU"
// @Param        customer  query     string  false  "Filter by customer name"
// @Success      200       {object}  envelope{data=[]domain.Inventory}
// @Failure      400       {object}  envelope{error=string}
// @Failure      500       {object}  envelope{error=string}
// @Router       /inventory [get]
func (handler *handler) listInventory(ctx fiber.Ctx) error {
	inventory, err := handler.service.ListInventory(ctx.Context(), service.InventoryFilters{
		Search:   ctx.Query("search"),
		SKU:      ctx.Query("sku"),
		Customer: ctx.Query("customer"),
	})
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: inventory})
}

// createInventory creates a new inventory master record.
//
// @Summary      Create inventory item
// @Description  Creates a new inventory record with initial physical stock.
// @Tags         Inventory
// @Accept       json
// @Produce      json
// @Param        request  body      createInventoryRequest  true  "Inventory payload"
// @Success      201      {object}  envelope{data=domain.Inventory}
// @Failure      400      {object}  envelope{error=string}
// @Failure      409      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /inventory [post]
func (handler *handler) createInventory(ctx fiber.Ctx) error {
	var request createInventoryRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return writeError(ctx, service.Validation("invalid inventory payload"))
	}

	inventory, err := handler.service.CreateInventory(ctx.Context(), service.CreateInventoryInput{
		SKU:           request.SKU,
		Name:          request.Name,
		CustomerName:  request.CustomerName,
		PhysicalStock: request.PhysicalStock,
	})
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(envelope{Data: inventory})
}

// adjustInventory creates an auditable stock adjustment transaction.
//
// @Summary      Adjust inventory stock
// @Description  Applies a physical stock correction as a dedicated adjustment transaction instead of a silent overwrite.
// @Tags         Inventory
// @Accept       json
// @Produce      json
// @Param        request  body      adjustInventoryRequest  true  "Adjustment payload"
// @Success      201      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      409      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /inventory/adjustments [post]
func (handler *handler) adjustInventory(ctx fiber.Ctx) error {
	var request adjustInventoryRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return writeError(ctx, service.Validation("invalid adjustment payload"))
	}

	transaction, err := handler.service.AdjustInventory(ctx.Context(), service.AdjustInventoryInput{
		InventoryID:      request.InventoryID,
		NewPhysicalStock: request.NewPhysicalStock,
		ReferenceCode:    request.ReferenceCode,
		Note:             request.Note,
	})
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(envelope{Data: transaction})
}

// createStockIn creates a stock-in transaction.
//
// @Summary      Create stock-in transaction
// @Description  Creates an inbound transaction in CREATED state with one or more line items.
// @Tags         Stock In
// @Accept       json
// @Produce      json
// @Param        request  body      createTransactionRequest  true  "Stock-in payload"
// @Success      201      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      409      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /stock-in [post]
func (handler *handler) createStockIn(ctx fiber.Ctx) error {
	request, err := parseCreateTransactionRequest(ctx)
	if err != nil {
		return writeError(ctx, err)
	}

	transaction, err := handler.service.CreateStockIn(ctx.Context(), request)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(envelope{Data: transaction})
}

// listStockIn lists stock-in transactions.
//
// @Summary      List stock-in transactions
// @Description  Returns inbound transactions and can filter by status.
// @Tags         Stock In
// @Produce      json
// @Param        status  query     string  false  "Optional stock-in status filter"  Enums(CREATED,IN_PROGRESS,DONE,CANCELLED)
// @Success      200     {object}  envelope{data=[]domain.Transaction}
// @Failure      400     {object}  envelope{error=string}
// @Failure      500     {object}  envelope{error=string}
// @Router       /stock-in [get]
func (handler *handler) listStockIn(ctx fiber.Ctx) error {
	transactions, err := handler.service.ListStockIn(ctx.Context(), ctx.Query("status"))
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: transactions})
}

// getStockIn returns a stock-in transaction by ID.
//
// @Summary      Get stock-in transaction
// @Description  Returns a single stock-in transaction including its items and status history.
// @Tags         Stock In
// @Produce      json
// @Param        id       path      int  true  "Transaction ID"
// @Success      200      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /stock-in/{id} [get]
func (handler *handler) getStockIn(ctx fiber.Ctx) error {
	id, err := parseID(ctx.Params("id"))
	if err != nil {
		return writeError(ctx, err)
	}

	transaction, err := handler.service.GetStockIn(ctx.Context(), id)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: transaction})
}

// updateStockInStatus advances a stock-in transaction.
//
// @Summary      Update stock-in status
// @Description  Advances a stock-in transaction to IN_PROGRESS or DONE.
// @Tags         Stock In
// @Accept       json
// @Produce      json
// @Param        id       path      int                  true  "Transaction ID"
// @Param        request  body      updateStatusRequest  true  "Next stock-in status"
// @Success      200      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      409      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /stock-in/{id}/status [patch]
func (handler *handler) updateStockInStatus(ctx fiber.Ctx) error {
	id, err := parseID(ctx.Params("id"))
	if err != nil {
		return writeError(ctx, err)
	}

	var request updateStatusRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return writeError(ctx, service.Validation("invalid stock-in status payload"))
	}

	transaction, err := handler.service.UpdateStockInStatus(ctx.Context(), id, request.Status, request.Note)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: transaction})
}

// cancelStockIn cancels a stock-in transaction.
//
// @Summary      Cancel stock-in transaction
// @Description  Cancels a stock-in transaction before it is completed.
// @Tags         Stock In
// @Accept       json
// @Produce      json
// @Param        id       path      int                     true   "Transaction ID"
// @Param        request  body      cancelTransactionRequest false  "Optional cancellation note"
// @Success      200      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      409      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /stock-in/{id}/cancel [post]
func (handler *handler) cancelStockIn(ctx fiber.Ctx) error {
	id, err := parseID(ctx.Params("id"))
	if err != nil {
		return writeError(ctx, err)
	}

	var request cancelTransactionRequest
	if ctx.HasBody() {
		if err := ctx.Bind().Body(&request); err != nil {
			return writeError(ctx, service.Validation("invalid stock-in cancellation payload"))
		}
	}

	transaction, err := handler.service.CancelStockIn(ctx.Context(), id, request.Note)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: transaction})
}

// createStockOut creates a stock-out transaction.
//
// @Summary      Create stock-out transaction
// @Description  Creates an outbound transaction in ALLOCATED state and reserves the requested stock.
// @Tags         Stock Out
// @Accept       json
// @Produce      json
// @Param        request  body      createTransactionRequest  true  "Stock-out payload"
// @Success      201      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      409      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /stock-out [post]
func (handler *handler) createStockOut(ctx fiber.Ctx) error {
	request, err := parseCreateTransactionRequest(ctx)
	if err != nil {
		return writeError(ctx, err)
	}

	transaction, err := handler.service.CreateStockOut(ctx.Context(), request)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.Status(fiber.StatusCreated).JSON(envelope{Data: transaction})
}

// listStockOut lists stock-out transactions.
//
// @Summary      List stock-out transactions
// @Description  Returns outbound transactions and can filter by status.
// @Tags         Stock Out
// @Produce      json
// @Param        status  query     string  false  "Optional stock-out status filter"  Enums(ALLOCATED,IN_PROGRESS,DONE,CANCELLED)
// @Success      200     {object}  envelope{data=[]domain.Transaction}
// @Failure      400     {object}  envelope{error=string}
// @Failure      500     {object}  envelope{error=string}
// @Router       /stock-out [get]
func (handler *handler) listStockOut(ctx fiber.Ctx) error {
	transactions, err := handler.service.ListStockOut(ctx.Context(), ctx.Query("status"))
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: transactions})
}

// getStockOut returns a stock-out transaction by ID.
//
// @Summary      Get stock-out transaction
// @Description  Returns a single stock-out transaction including items and history.
// @Tags         Stock Out
// @Produce      json
// @Param        id       path      int  true  "Transaction ID"
// @Success      200      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /stock-out/{id} [get]
func (handler *handler) getStockOut(ctx fiber.Ctx) error {
	id, err := parseID(ctx.Params("id"))
	if err != nil {
		return writeError(ctx, err)
	}

	transaction, err := handler.service.GetStockOut(ctx.Context(), id)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: transaction})
}

// updateStockOutStatus advances a stock-out transaction.
//
// @Summary      Update stock-out status
// @Description  Advances a stock-out transaction to IN_PROGRESS or DONE.
// @Tags         Stock Out
// @Accept       json
// @Produce      json
// @Param        id       path      int                  true  "Transaction ID"
// @Param        request  body      updateStatusRequest  true  "Next stock-out status"
// @Success      200      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      409      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /stock-out/{id}/status [patch]
func (handler *handler) updateStockOutStatus(ctx fiber.Ctx) error {
	id, err := parseID(ctx.Params("id"))
	if err != nil {
		return writeError(ctx, err)
	}

	var request updateStatusRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return writeError(ctx, service.Validation("invalid stock-out status payload"))
	}

	transaction, err := handler.service.UpdateStockOutStatus(ctx.Context(), id, request.Status, request.Note)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: transaction})
}

// cancelStockOut cancels a stock-out transaction.
//
// @Summary      Cancel stock-out transaction
// @Description  Cancels a stock-out transaction and releases reserved stock.
// @Tags         Stock Out
// @Accept       json
// @Produce      json
// @Param        id       path      int                      true   "Transaction ID"
// @Param        request  body      cancelTransactionRequest false  "Optional cancellation note"
// @Success      200      {object}  envelope{data=domain.Transaction}
// @Failure      400      {object}  envelope{error=string}
// @Failure      404      {object}  envelope{error=string}
// @Failure      409      {object}  envelope{error=string}
// @Failure      500      {object}  envelope{error=string}
// @Router       /stock-out/{id}/cancel [post]
func (handler *handler) cancelStockOut(ctx fiber.Ctx) error {
	id, err := parseID(ctx.Params("id"))
	if err != nil {
		return writeError(ctx, err)
	}

	var request cancelTransactionRequest
	if ctx.HasBody() {
		if err := ctx.Bind().Body(&request); err != nil {
			return writeError(ctx, service.Validation("invalid stock-out cancellation payload"))
		}
	}

	transaction, err := handler.service.CancelStockOut(ctx.Context(), id, request.Note)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: transaction})
}

// listReports returns completed stock movement reports.
//
// @Summary      List completed movement reports
// @Description  Returns only DONE stock-in and DONE stock-out transactions for reporting, with paginated transaction-level detail and filter support.
// @Tags         Reports
// @Produce      json
// @Param        type           query     string  false  "Optional transaction type filter"      Enums(STOCK_IN,STOCK_OUT)
// @Param        referenceCode  query     string  false  "Partial reference code filter"
// @Param        completedFrom  query     string  false  "Inclusive completed date filter"      Format(date)
// @Param        completedTo    query     string  false  "Inclusive completed end date filter"  Format(date)
// @Param        limit   query     int  false  "Max number of report transactions to return"  minimum(1) maximum(100)
// @Param        offset  query     int  false  "Zero-based offset for report pagination"      minimum(0)
// @Success      200     {object}  envelope{data=reportListResponse}
// @Failure      400     {object}  envelope{error=string}
// @Failure      500     {object}  envelope{error=string}
// @Router       /reports [get]
func (handler *handler) listReports(ctx fiber.Ctx) error {
	filters, err := parseReportFilters(ctx)
	if err != nil {
		return writeError(ctx, err)
	}

	reports, err := handler.service.ListReports(ctx.Context(), filters)
	if err != nil {
		return writeError(ctx, err)
	}

	return ctx.JSON(envelope{Data: reportListResponse{
		Items:    reports.Items,
		Total:    reports.Total,
		Limit:    reports.Limit,
		Offset:   reports.Offset,
		UnitsIn:  reports.UnitsIn,
		UnitsOut: reports.UnitsOut,
	}})
}

// exportReports returns completed stock movement reports as a CSV attachment.
//
// @Summary      Export completed movement reports as CSV
// @Description  Exports all DONE stock-in and DONE stock-out transactions matching the provided filters as a CSV download.
// @Tags         Reports
// @Produce      text/csv
// @Param        type           query     string  false  "Optional transaction type filter"      Enums(STOCK_IN,STOCK_OUT)
// @Param        referenceCode  query     string  false  "Partial reference code filter"
// @Param        completedFrom  query     string  false  "Inclusive completed date filter"      Format(date)
// @Param        completedTo    query     string  false  "Inclusive completed end date filter"  Format(date)
// @Success      200            {string}  string  "CSV file"
// @Failure      400            {object}  envelope{error=string}
// @Failure      500            {object}  envelope{error=string}
// @Router       /reports/export [get]
func (handler *handler) exportReports(ctx fiber.Ctx) error {
	filters, err := parseReportFilters(ctx)
	if err != nil {
		return writeError(ctx, err)
	}

	reports, err := handler.service.ListAllReports(ctx.Context(), filters)
	if err != nil {
		return writeError(ctx, err)
	}

	csvPayload, err := buildReportCSV(reports)
	if err != nil {
		return writeError(ctx, fmt.Errorf("build report csv: %w", err))
	}

	ctx.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	ctx.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", reportExportFilename()))

	return ctx.Send(csvPayload)
}

func parseCreateTransactionRequest(ctx fiber.Ctx) (service.CreateTransactionInput, error) {
	var request createTransactionRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return service.CreateTransactionInput{}, service.Validation("invalid transaction payload")
	}

	items := make([]service.TransactionItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, service.TransactionItemInput{
			InventoryID: item.InventoryID,
			Quantity:    item.Quantity,
		})
	}

	return service.CreateTransactionInput{
		ReferenceCode: request.ReferenceCode,
		Note:          request.Note,
		Items:         items,
	}, nil
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, service.Validation("id must be a positive integer")
	}

	return id, nil
}

func parseReportFilters(ctx fiber.Ctx) (service.ReportFilters, error) {
	filters := service.ReportFilters{}

	if rawType := ctx.Query("type"); rawType != "" {
		reportType, err := domain.ParseTransactionType(rawType)
		if err != nil {
			return service.ReportFilters{}, service.Validation("type must be STOCK_IN or STOCK_OUT")
		}
		filters.Type = &reportType
	}

	filters.ReferenceCode = ctx.Query("referenceCode")

	if rawCompletedFrom := ctx.Query("completedFrom"); rawCompletedFrom != "" {
		completedFrom, err := time.Parse("2006-01-02", rawCompletedFrom)
		if err != nil {
			return service.ReportFilters{}, service.Validation("completedFrom must use YYYY-MM-DD format")
		}
		completedFrom = completedFrom.UTC()
		filters.CompletedFrom = &completedFrom
	}

	if rawCompletedTo := ctx.Query("completedTo"); rawCompletedTo != "" {
		completedTo, err := time.Parse("2006-01-02", rawCompletedTo)
		if err != nil {
			return service.ReportFilters{}, service.Validation("completedTo must use YYYY-MM-DD format")
		}
		completedTo = completedTo.UTC().Add(24 * time.Hour)
		filters.CompletedTo = &completedTo
	}

	if rawLimit := ctx.Query("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return service.ReportFilters{}, service.Validation("limit must be a positive integer")
		}
		filters.Limit = limit
	}

	if rawOffset := ctx.Query("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return service.ReportFilters{}, service.Validation("offset must be a non-negative integer")
		}
		filters.Offset = offset
	}

	return filters, nil
}

func buildReportCSV(reports []domain.Transaction) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	if err := writer.Write([]string{
		"transaction_id",
		"reference_code",
		"transaction_type",
		"status",
		"created_at",
		"completed_at",
		"operator_note",
		"inventory_id",
		"sku",
		"item_name",
		"customer_name",
		"quantity",
		"status_history",
	}); err != nil {
		return nil, err
	}

	for _, report := range reports {
		historySummary := make([]string, 0, len(report.History))
		for _, entry := range report.History {
			historySummary = append(historySummary, fmt.Sprintf("%s @ %s (%s)", entry.Status, entry.CreatedAt.Format(time.RFC3339), entry.Note))
		}

		for _, item := range report.Items {
			if err := writer.Write([]string{
				strconv.FormatInt(report.ID, 10),
				report.ReferenceCode,
				string(report.Type),
				string(report.Status),
				report.CreatedAt.Format(time.RFC3339),
				formatCSVDate(report.CompletedAt),
				report.Note,
				strconv.FormatInt(item.InventoryID, 10),
				item.SKU,
				item.Name,
				item.CustomerName,
				strconv.FormatInt(item.Quantity, 10),
				joinCSVHistory(historySummary),
			}); err != nil {
				return nil, err
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func formatCSVDate(value *time.Time) string {
	if value == nil {
		return ""
	}

	return value.UTC().Format(time.RFC3339)
}

func joinCSVHistory(entries []string) string {
	if len(entries) == 0 {
		return ""
	}

	return fmt.Sprintf("%s", bytes.Join(func() [][]byte {
		joined := make([][]byte, 0, len(entries))
		for _, entry := range entries {
			joined = append(joined, []byte(entry))
		}
		return joined
	}(), []byte(" | ")))
}

func reportExportFilename() string {
	return fmt.Sprintf("smart-inventory-report-%s.csv", time.Now().UTC().Format("20060102-150405"))
}

func writeError(ctx fiber.Ctx, err error) error {
	statusCode := fiber.StatusInternalServerError

	switch {
	case errors.Is(err, service.ErrValidation):
		statusCode = fiber.StatusBadRequest
	case errors.Is(err, service.ErrNotFound):
		statusCode = fiber.StatusNotFound
	case errors.Is(err, service.ErrConflict):
		statusCode = fiber.StatusConflict
	case errors.Is(err, service.ErrInsufficientStock):
		statusCode = fiber.StatusConflict
	}

	return ctx.Status(statusCode).JSON(envelope{Error: err.Error()})
}
