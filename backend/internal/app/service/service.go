package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wecrazy/smart-inventory-core-system/backend/internal/domain"
)

type InventoryFilters struct {
	Search   string
	SKU      string
	Customer string
}

type CreateInventoryInput struct {
	SKU           string
	Name          string
	CustomerName  string
	PhysicalStock int64
}

type AdjustInventoryInput struct {
	InventoryID      int64
	NewPhysicalStock int64
	ReferenceCode    string
	Note             string
}

type TransactionItemInput struct {
	InventoryID int64
	Quantity    int64
}

type CreateTransactionInput struct {
	ReferenceCode string
	Note          string
	Items         []TransactionItemInput
}

type ReportFilters struct {
	Limit         int
	Offset        int
	Type          *domain.TransactionType
	ReferenceCode string
	CompletedFrom *time.Time
	CompletedTo   *time.Time
}

type ReportPage struct {
	Items    []domain.Transaction
	Total    int64
	Limit    int
	Offset   int
	UnitsIn  int64
	UnitsOut int64
}

const (
	DefaultReportLimit = 10
	MaxReportLimit     = 100
)

type InventoryStore interface {
	ListInventory(ctx context.Context, filters InventoryFilters) ([]domain.Inventory, error)
	CreateInventory(ctx context.Context, input CreateInventoryInput) (domain.Inventory, error)
	AdjustInventory(ctx context.Context, input AdjustInventoryInput) (domain.Transaction, error)
}

type TransactionStore interface {
	CreateStockIn(ctx context.Context, input CreateTransactionInput) (domain.Transaction, error)
	UpdateStockInStatus(ctx context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error)
	CancelStockIn(ctx context.Context, transactionID int64, note string) (domain.Transaction, error)
	CreateStockOut(ctx context.Context, input CreateTransactionInput) (domain.Transaction, error)
	UpdateStockOutStatus(ctx context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error)
	CancelStockOut(ctx context.Context, transactionID int64, note string) (domain.Transaction, error)
	GetTransaction(ctx context.Context, transactionID int64) (domain.Transaction, error)
	ListTransactions(ctx context.Context, transactionType domain.TransactionType, status *domain.TransactionStatus) ([]domain.Transaction, error)
	ListReports(ctx context.Context, filters ReportFilters) (ReportPage, error)
	ListAllReports(ctx context.Context, filters ReportFilters) ([]domain.Transaction, error)
}

type Service struct {
	inventoryStore   InventoryStore
	transactionStore TransactionStore
}

func New(inventoryStore InventoryStore, transactionStore TransactionStore) *Service {
	return &Service{
		inventoryStore:   inventoryStore,
		transactionStore: transactionStore,
	}
}

func (service *Service) ListInventory(ctx context.Context, filters InventoryFilters) ([]domain.Inventory, error) {
	filters.Search = strings.TrimSpace(filters.Search)
	filters.SKU = strings.TrimSpace(strings.ToUpper(filters.SKU))
	filters.Customer = strings.TrimSpace(filters.Customer)

	return service.inventoryStore.ListInventory(ctx, filters)
}

func (service *Service) CreateInventory(ctx context.Context, input CreateInventoryInput) (domain.Inventory, error) {
	input.SKU = strings.ToUpper(strings.TrimSpace(input.SKU))
	input.Name = strings.TrimSpace(input.Name)
	input.CustomerName = strings.TrimSpace(input.CustomerName)

	if input.SKU == "" {
		return domain.Inventory{}, Validation("sku is required")
	}
	if input.Name == "" {
		return domain.Inventory{}, Validation("name is required")
	}
	if input.CustomerName == "" {
		return domain.Inventory{}, Validation("customer name is required")
	}
	if input.PhysicalStock < 0 {
		return domain.Inventory{}, Validation("physical stock cannot be negative")
	}

	return service.inventoryStore.CreateInventory(ctx, input)
}

func (service *Service) AdjustInventory(ctx context.Context, input AdjustInventoryInput) (domain.Transaction, error) {
	input.Note = strings.TrimSpace(input.Note)
	if input.InventoryID <= 0 {
		return domain.Transaction{}, Validation("inventoryId must be greater than zero")
	}
	if input.NewPhysicalStock < 0 {
		return domain.Transaction{}, Validation("newPhysicalStock cannot be negative")
	}

	input.ReferenceCode = service.referenceCode(input.ReferenceCode, domain.TypeAdjustment)

	return service.inventoryStore.AdjustInventory(ctx, input)
}

func (service *Service) CreateStockIn(ctx context.Context, input CreateTransactionInput) (domain.Transaction, error) {
	normalized, err := service.normalizeTransactionInput(input, domain.TypeStockIn)
	if err != nil {
		return domain.Transaction{}, err
	}

	return service.transactionStore.CreateStockIn(ctx, normalized)
}

func (service *Service) ListStockIn(ctx context.Context, rawStatus string) ([]domain.Transaction, error) {
	status, err := parseOptionalStatus(rawStatus)
	if err != nil {
		return nil, err
	}

	return service.transactionStore.ListTransactions(ctx, domain.TypeStockIn, status)
}

func (service *Service) GetStockIn(ctx context.Context, transactionID int64) (domain.Transaction, error) {
	transaction, err := service.transactionStore.GetTransaction(ctx, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if transaction.Type != domain.TypeStockIn {
		return domain.Transaction{}, NotFound("stock-in transaction not found")
	}

	return transaction, nil
}

func (service *Service) UpdateStockInStatus(ctx context.Context, transactionID int64, rawStatus string, note string) (domain.Transaction, error) {
	status, err := domain.ParseTransactionStatus(rawStatus)
	if err != nil {
		return domain.Transaction{}, Validation(err.Error())
	}
	if status != domain.StatusInProgress && status != domain.StatusDone {
		return domain.Transaction{}, Validation("stock-in status endpoint only accepts IN_PROGRESS or DONE")
	}

	return service.transactionStore.UpdateStockInStatus(ctx, transactionID, status, strings.TrimSpace(note))
}

func (service *Service) CancelStockIn(ctx context.Context, transactionID int64, note string) (domain.Transaction, error) {
	return service.transactionStore.CancelStockIn(ctx, transactionID, strings.TrimSpace(note))
}

func (service *Service) CreateStockOut(ctx context.Context, input CreateTransactionInput) (domain.Transaction, error) {
	normalized, err := service.normalizeTransactionInput(input, domain.TypeStockOut)
	if err != nil {
		return domain.Transaction{}, err
	}

	return service.transactionStore.CreateStockOut(ctx, normalized)
}

func (service *Service) ListStockOut(ctx context.Context, rawStatus string) ([]domain.Transaction, error) {
	status, err := parseOptionalStatus(rawStatus)
	if err != nil {
		return nil, err
	}

	return service.transactionStore.ListTransactions(ctx, domain.TypeStockOut, status)
}

func (service *Service) GetStockOut(ctx context.Context, transactionID int64) (domain.Transaction, error) {
	transaction, err := service.transactionStore.GetTransaction(ctx, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if transaction.Type != domain.TypeStockOut {
		return domain.Transaction{}, NotFound("stock-out transaction not found")
	}

	return transaction, nil
}

func (service *Service) UpdateStockOutStatus(ctx context.Context, transactionID int64, rawStatus string, note string) (domain.Transaction, error) {
	status, err := domain.ParseTransactionStatus(rawStatus)
	if err != nil {
		return domain.Transaction{}, Validation(err.Error())
	}
	if status != domain.StatusInProgress && status != domain.StatusDone {
		return domain.Transaction{}, Validation("stock-out status endpoint only accepts IN_PROGRESS or DONE")
	}

	return service.transactionStore.UpdateStockOutStatus(ctx, transactionID, status, strings.TrimSpace(note))
}

func (service *Service) CancelStockOut(ctx context.Context, transactionID int64, note string) (domain.Transaction, error) {
	return service.transactionStore.CancelStockOut(ctx, transactionID, strings.TrimSpace(note))
}

func (service *Service) ListReports(ctx context.Context, filters ReportFilters) (ReportPage, error) {
	filters, err := service.normalizeReportFilters(filters, true)
	if err != nil {
		return ReportPage{}, err
	}

	return service.transactionStore.ListReports(ctx, filters)
}

func (service *Service) ListAllReports(ctx context.Context, filters ReportFilters) ([]domain.Transaction, error) {
	filters, err := service.normalizeReportFilters(filters, false)
	if err != nil {
		return nil, err
	}

	return service.transactionStore.ListAllReports(ctx, filters)
}

func (service *Service) normalizeReportFilters(filters ReportFilters, paged bool) (ReportFilters, error) {
	if filters.Limit <= 0 {
		filters.Limit = DefaultReportLimit
	}
	if filters.Limit > MaxReportLimit {
		filters.Limit = MaxReportLimit
	}
	if filters.Offset < 0 {
		return ReportFilters{}, Validation("offset cannot be negative")
	}
	if !paged {
		filters.Limit = 0
		filters.Offset = 0
	}

	filters.ReferenceCode = strings.TrimSpace(strings.ToUpper(filters.ReferenceCode))

	if filters.Type != nil && *filters.Type != domain.TypeStockIn && *filters.Type != domain.TypeStockOut {
		return ReportFilters{}, Validation("report type filter only accepts STOCK_IN or STOCK_OUT")
	}
	if filters.CompletedFrom != nil && filters.CompletedTo != nil && filters.CompletedTo.Before(*filters.CompletedFrom) {
		return ReportFilters{}, Validation("completedTo cannot be earlier than completedFrom")
	}

	return filters, nil
}

func (service *Service) normalizeTransactionInput(input CreateTransactionInput, transactionType domain.TransactionType) (CreateTransactionInput, error) {
	input.ReferenceCode = service.referenceCode(input.ReferenceCode, transactionType)
	input.Note = strings.TrimSpace(input.Note)

	if len(input.Items) == 0 {
		return CreateTransactionInput{}, Validation("at least one item is required")
	}

	mergedItems := make(map[int64]int64, len(input.Items))
	for _, item := range input.Items {
		if item.InventoryID <= 0 {
			return CreateTransactionInput{}, Validation("inventoryId must be greater than zero")
		}
		if item.Quantity <= 0 {
			return CreateTransactionInput{}, Validation("quantity must be greater than zero")
		}
		mergedItems[item.InventoryID] += item.Quantity
	}

	normalized := make([]TransactionItemInput, 0, len(mergedItems))
	for inventoryID, quantity := range mergedItems {
		normalized = append(normalized, TransactionItemInput{InventoryID: inventoryID, Quantity: quantity})
	}

	input.Items = normalized

	return input, nil
}

func (service *Service) referenceCode(candidate string, transactionType domain.TransactionType) string {
	candidate = strings.TrimSpace(strings.ToUpper(candidate))
	if candidate != "" {
		return candidate
	}

	return fmt.Sprintf("%s-%s", transactionType.ReferencePrefix(), time.Now().UTC().Format("20060102-150405.000000"))
}

func parseOptionalStatus(raw string) (*domain.TransactionStatus, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	status, err := domain.ParseTransactionStatus(raw)
	if err != nil {
		return nil, Validation(err.Error())
	}

	return &status, nil
}
