package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wecrazy/smart-inventory-core-system/backend/internal/app/service"
	"github.com/wecrazy/smart-inventory-core-system/backend/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

type queryable interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) ListInventory(ctx context.Context, filters service.InventoryFilters) ([]domain.Inventory, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id, sku, name, customer_name, physical_stock, reserved_stock, created_at, updated_at
		FROM inventory_items
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%' OR sku ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR sku ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR customer_name ILIKE '%' || $3 || '%')
		ORDER BY updated_at DESC, id DESC
	`, filters.Search, filters.SKU, filters.Customer)
	if err != nil {
		return nil, fmt.Errorf("list inventory: %w", err)
	}
	defer rows.Close()

	inventory := make([]domain.Inventory, 0)
	for rows.Next() {
		var item domain.Inventory
		if err := rows.Scan(
			&item.ID,
			&item.SKU,
			&item.Name,
			&item.CustomerName,
			&item.PhysicalStock,
			&item.ReservedStock,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan inventory: %w", err)
		}
		item.RefreshAvailable()
		inventory = append(inventory, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory: %w", err)
	}

	return inventory, nil
}

func (repository *Repository) CreateInventory(ctx context.Context, input service.CreateInventoryInput) (domain.Inventory, error) {
	var item domain.Inventory
	if err := repository.pool.QueryRow(ctx, `
		INSERT INTO inventory_items (sku, name, customer_name, physical_stock, reserved_stock)
		VALUES ($1, $2, $3, $4, 0)
		RETURNING id, sku, name, customer_name, physical_stock, reserved_stock, created_at, updated_at
	`, input.SKU, input.Name, input.CustomerName, input.PhysicalStock).Scan(
		&item.ID,
		&item.SKU,
		&item.Name,
		&item.CustomerName,
		&item.PhysicalStock,
		&item.ReservedStock,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if uniqueViolation(err) {
			return domain.Inventory{}, service.Conflict("sku already exists")
		}

		return domain.Inventory{}, fmt.Errorf("create inventory: %w", err)
	}
	item.RefreshAvailable()

	return item, nil
}

func (repository *Repository) AdjustInventory(ctx context.Context, input service.AdjustInventoryInput) (domain.Transaction, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("begin adjustment transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var inventory domain.Inventory
	if err := tx.QueryRow(ctx, `
		SELECT id, sku, name, customer_name, physical_stock, reserved_stock, created_at, updated_at
		FROM inventory_items
		WHERE id = $1
		FOR UPDATE
	`, input.InventoryID).Scan(
		&inventory.ID,
		&inventory.SKU,
		&inventory.Name,
		&inventory.CustomerName,
		&inventory.PhysicalStock,
		&inventory.ReservedStock,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Transaction{}, service.NotFound("inventory item not found")
		}

		return domain.Transaction{}, fmt.Errorf("lock inventory item: %w", err)
	}

	if input.NewPhysicalStock < inventory.ReservedStock {
		return domain.Transaction{}, service.Conflict("new physical stock cannot be lower than reserved stock")
	}

	adjustmentDelta := input.NewPhysicalStock - inventory.PhysicalStock

	if _, err := tx.Exec(ctx, `
		UPDATE inventory_items
		SET physical_stock = $2, updated_at = NOW()
		WHERE id = $1
	`, inventory.ID, input.NewPhysicalStock); err != nil {
		return domain.Transaction{}, fmt.Errorf("update inventory stock: %w", err)
	}

	transaction, err := repository.insertTransaction(ctx, tx, domain.TypeAdjustment, domain.StatusDone, input.ReferenceCode, input.Note)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := repository.insertTransactionItem(ctx, tx, transaction.ID, domain.TransactionItem{
		InventoryID:  inventory.ID,
		SKU:          inventory.SKU,
		Name:         inventory.Name,
		CustomerName: inventory.CustomerName,
		Quantity:     adjustmentDelta,
	}); err != nil {
		return domain.Transaction{}, err
	}

	if err := repository.insertHistory(ctx, tx, transaction.ID, domain.StatusDone, input.Note); err != nil {
		return domain.Transaction{}, err
	}

	loadedTransaction, err := repository.loadTransaction(ctx, tx, transaction.ID)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Transaction{}, fmt.Errorf("commit adjustment transaction: %w", err)
	}

	return loadedTransaction, nil
}

func (repository *Repository) CreateStockIn(ctx context.Context, input service.CreateTransactionInput) (domain.Transaction, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("begin stock-in transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	transaction, err := repository.insertTransaction(ctx, tx, domain.TypeStockIn, domain.StatusCreated, input.ReferenceCode, input.Note)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := repository.insertItemsFromInput(ctx, tx, transaction.ID, input.Items); err != nil {
		return domain.Transaction{}, err
	}

	if err := repository.insertHistory(ctx, tx, transaction.ID, domain.StatusCreated, input.Note); err != nil {
		return domain.Transaction{}, err
	}

	loadedTransaction, err := repository.loadTransaction(ctx, tx, transaction.ID)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Transaction{}, fmt.Errorf("commit stock-in transaction: %w", err)
	}

	return loadedTransaction, nil
}

func (repository *Repository) UpdateStockInStatus(ctx context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("begin stock-in status transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	transaction, err := repository.lockTransaction(ctx, tx, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if transaction.Type != domain.TypeStockIn {
		return domain.Transaction{}, service.NotFound("stock-in transaction not found")
	}
	if err := domain.ValidateStockInTransition(transaction.Status, next); err != nil {
		return domain.Transaction{}, service.Conflict(err.Error())
	}

	if next == domain.StatusDone {
		items, err := repository.loadTransactionItems(ctx, tx, transaction.ID)
		if err != nil {
			return domain.Transaction{}, err
		}
		for _, item := range items {
			if _, err := tx.Exec(ctx, `
				UPDATE inventory_items
				SET physical_stock = physical_stock + $2, updated_at = NOW()
				WHERE id = $1
			`, item.InventoryID, item.Quantity); err != nil {
				return domain.Transaction{}, fmt.Errorf("apply stock-in item: %w", err)
			}
		}
	}

	if err := repository.updateTransactionStatus(ctx, tx, transaction.ID, next, note); err != nil {
		return domain.Transaction{}, err
	}

	if err := repository.insertHistory(ctx, tx, transaction.ID, next, note); err != nil {
		return domain.Transaction{}, err
	}

	loadedTransaction, err := repository.loadTransaction(ctx, tx, transaction.ID)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Transaction{}, fmt.Errorf("commit stock-in status transaction: %w", err)
	}

	return loadedTransaction, nil
}

func (repository *Repository) CancelStockIn(ctx context.Context, transactionID int64, note string) (domain.Transaction, error) {
	return repository.UpdateStockInStatus(ctx, transactionID, domain.StatusCancelled, note)
}

func (repository *Repository) CreateStockOut(ctx context.Context, input service.CreateTransactionInput) (domain.Transaction, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("begin stock-out transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	transaction, err := repository.insertTransaction(ctx, tx, domain.TypeStockOut, domain.StatusAllocated, input.ReferenceCode, input.Note)
	if err != nil {
		return domain.Transaction{}, err
	}

	for _, requestItem := range input.Items {
		inventory, err := repository.lockInventory(ctx, tx, requestItem.InventoryID)
		if err != nil {
			return domain.Transaction{}, err
		}
		if inventory.AvailableStock < requestItem.Quantity {
			return domain.Transaction{}, service.InsufficientStock(fmt.Sprintf("insufficient available stock for sku %s", inventory.SKU))
		}

		if _, err := tx.Exec(ctx, `
			UPDATE inventory_items
			SET reserved_stock = reserved_stock + $2, updated_at = NOW()
			WHERE id = $1
		`, inventory.ID, requestItem.Quantity); err != nil {
			return domain.Transaction{}, fmt.Errorf("reserve stock: %w", err)
		}

		item := domain.TransactionItem{
			InventoryID:  inventory.ID,
			SKU:          inventory.SKU,
			Name:         inventory.Name,
			CustomerName: inventory.CustomerName,
			Quantity:     requestItem.Quantity,
		}
		if err := repository.insertTransactionItem(ctx, tx, transaction.ID, item); err != nil {
			return domain.Transaction{}, err
		}
		if err := repository.insertReservation(ctx, tx, transaction.ID, item.InventoryID, item.Quantity); err != nil {
			return domain.Transaction{}, err
		}
	}

	if err := repository.insertHistory(ctx, tx, transaction.ID, domain.StatusAllocated, input.Note); err != nil {
		return domain.Transaction{}, err
	}

	loadedTransaction, err := repository.loadTransaction(ctx, tx, transaction.ID)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Transaction{}, fmt.Errorf("commit stock-out transaction: %w", err)
	}

	return loadedTransaction, nil
}

func (repository *Repository) UpdateStockOutStatus(ctx context.Context, transactionID int64, next domain.TransactionStatus, note string) (domain.Transaction, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("begin stock-out status transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	transaction, err := repository.lockTransaction(ctx, tx, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if transaction.Type != domain.TypeStockOut {
		return domain.Transaction{}, service.NotFound("stock-out transaction not found")
	}
	if err := domain.ValidateStockOutTransition(transaction.Status, next); err != nil {
		return domain.Transaction{}, service.Conflict(err.Error())
	}

	if next == domain.StatusDone {
		reservations, err := repository.loadReservations(ctx, tx, transaction.ID, domain.ReservationActive)
		if err != nil {
			return domain.Transaction{}, err
		}
		for _, reservation := range reservations {
			inventory, err := repository.lockInventory(ctx, tx, reservation.InventoryID)
			if err != nil {
				return domain.Transaction{}, err
			}
			if inventory.ReservedStock < reservation.Quantity {
				return domain.Transaction{}, service.Conflict("reserved stock is lower than reservation quantity")
			}

			if _, err := tx.Exec(ctx, `
				UPDATE inventory_items
				SET physical_stock = physical_stock - $2,
				    reserved_stock = reserved_stock - $2,
				    updated_at = NOW()
				WHERE id = $1
			`, inventory.ID, reservation.Quantity); err != nil {
				return domain.Transaction{}, fmt.Errorf("finalize stock-out inventory update: %w", err)
			}

			if _, err := tx.Exec(ctx, `
				UPDATE stock_reservations
				SET status = $2, updated_at = NOW()
				WHERE transaction_id = $1 AND inventory_item_id = $3 AND status = $4
			`, transaction.ID, domain.ReservationFulfilled, reservation.InventoryID, domain.ReservationActive); err != nil {
				return domain.Transaction{}, fmt.Errorf("mark reservation fulfilled: %w", err)
			}
		}
	}

	if err := repository.updateTransactionStatus(ctx, tx, transaction.ID, next, note); err != nil {
		return domain.Transaction{}, err
	}

	if err := repository.insertHistory(ctx, tx, transaction.ID, next, note); err != nil {
		return domain.Transaction{}, err
	}

	loadedTransaction, err := repository.loadTransaction(ctx, tx, transaction.ID)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Transaction{}, fmt.Errorf("commit stock-out status transaction: %w", err)
	}

	return loadedTransaction, nil
}

func (repository *Repository) CancelStockOut(ctx context.Context, transactionID int64, note string) (domain.Transaction, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("begin stock-out cancel transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	transaction, err := repository.lockTransaction(ctx, tx, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if transaction.Type != domain.TypeStockOut {
		return domain.Transaction{}, service.NotFound("stock-out transaction not found")
	}
	if err := domain.ValidateStockOutTransition(transaction.Status, domain.StatusCancelled); err != nil {
		return domain.Transaction{}, service.Conflict(err.Error())
	}

	reservations, err := repository.loadReservations(ctx, tx, transaction.ID, domain.ReservationActive)
	if err != nil {
		return domain.Transaction{}, err
	}
	for _, reservation := range reservations {
		inventory, err := repository.lockInventory(ctx, tx, reservation.InventoryID)
		if err != nil {
			return domain.Transaction{}, err
		}
		if inventory.ReservedStock < reservation.Quantity {
			return domain.Transaction{}, service.Conflict("reserved stock is lower than reservation quantity")
		}

		if _, err := tx.Exec(ctx, `
			UPDATE inventory_items
			SET reserved_stock = reserved_stock - $2, updated_at = NOW()
			WHERE id = $1
		`, inventory.ID, reservation.Quantity); err != nil {
			return domain.Transaction{}, fmt.Errorf("release reserved stock: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE stock_reservations
			SET status = $2, updated_at = NOW()
			WHERE transaction_id = $1 AND inventory_item_id = $3 AND status = $4
		`, transaction.ID, domain.ReservationReleased, reservation.InventoryID, domain.ReservationActive); err != nil {
			return domain.Transaction{}, fmt.Errorf("mark reservation released: %w", err)
		}
	}

	if err := repository.updateTransactionStatus(ctx, tx, transaction.ID, domain.StatusCancelled, note); err != nil {
		return domain.Transaction{}, err
	}

	if err := repository.insertHistory(ctx, tx, transaction.ID, domain.StatusCancelled, note); err != nil {
		return domain.Transaction{}, err
	}

	loadedTransaction, err := repository.loadTransaction(ctx, tx, transaction.ID)
	if err != nil {
		return domain.Transaction{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Transaction{}, fmt.Errorf("commit stock-out cancel transaction: %w", err)
	}

	return loadedTransaction, nil
}

func (repository *Repository) GetTransaction(ctx context.Context, transactionID int64) (domain.Transaction, error) {
	return repository.loadTransaction(ctx, repository.pool, transactionID)
}

func (repository *Repository) ListTransactions(ctx context.Context, transactionType domain.TransactionType, status *domain.TransactionStatus) ([]domain.Transaction, error) {
	statusFilter := ""
	if status != nil {
		statusFilter = string(*status)
	}

	rows, err := repository.pool.Query(ctx, `
		SELECT id, type, status, reference_code, note, completed_at, created_at, updated_at
		FROM stock_transactions
		WHERE type = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC, id DESC
	`, transactionType, statusFilter)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]domain.Transaction, 0)
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}

		transaction.Items, err = repository.loadTransactionItems(ctx, repository.pool, transaction.ID)
		if err != nil {
			return nil, err
		}
		transaction.History, err = repository.loadHistory(ctx, repository.pool, transaction.ID)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}

	return transactions, nil
}

func (repository *Repository) ListReports(ctx context.Context, filters service.ReportFilters) (service.ReportPage, error) {
	countArgs, completedFrom, completedTo, referenceCode, typeFilter := reportQueryArgs(filters)

	var total int64
	if err := repository.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM stock_transactions
		WHERE status = $1
		  AND type IN ($2, $3)
		  AND ($4 = '' OR type = $4)
		  AND ($5 = '' OR reference_code ILIKE '%' || $5 || '%')
		  AND ($6::timestamptz IS NULL OR completed_at >= $6::timestamptz)
		  AND ($7::timestamptz IS NULL OR completed_at < $7::timestamptz)
	`, countArgs...).Scan(&total); err != nil {
		return service.ReportPage{}, fmt.Errorf("count reports: %w", err)
	}

	var unitsIn, unitsOut int64
	aggregateArgs := []any{domain.StatusDone, domain.TypeStockIn, domain.TypeStockOut, typeFilter, referenceCode, completedFrom, completedTo}
	if err := repository.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN tx.type = $2 THEN item.quantity ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN tx.type = $3 THEN item.quantity ELSE 0 END), 0)
		FROM stock_transactions tx
		JOIN stock_transaction_items item ON item.transaction_id = tx.id
		WHERE tx.status = $1
		  AND tx.type IN ($2, $3)
		  AND ($4 = '' OR tx.type = $4)
		  AND ($5 = '' OR tx.reference_code ILIKE '%' || $5 || '%')
		  AND ($6::timestamptz IS NULL OR tx.completed_at >= $6::timestamptz)
		  AND ($7::timestamptz IS NULL OR tx.completed_at < $7::timestamptz)
	`, aggregateArgs...).Scan(&unitsIn, &unitsOut); err != nil {
		return service.ReportPage{}, fmt.Errorf("aggregate reports: %w", err)
	}

	rows, err := repository.pool.Query(ctx, `
		SELECT id, type, status, reference_code, note, completed_at, created_at, updated_at
		FROM stock_transactions
		WHERE status = $1
		  AND type IN ($2, $3)
		  AND ($4 = '' OR type = $4)
		  AND ($5 = '' OR reference_code ILIKE '%' || $5 || '%')
		  AND ($6::timestamptz IS NULL OR completed_at >= $6::timestamptz)
		  AND ($7::timestamptz IS NULL OR completed_at < $7::timestamptz)
		ORDER BY completed_at DESC NULLS LAST, created_at DESC, id DESC
		LIMIT $8 OFFSET $9
	`, domain.StatusDone, domain.TypeStockIn, domain.TypeStockOut, typeFilter, referenceCode, completedFrom, completedTo, filters.Limit, filters.Offset)
	if err != nil {
		return service.ReportPage{}, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	reports := make([]domain.Transaction, 0)
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return service.ReportPage{}, err
		}

		transaction.Items, err = repository.loadTransactionItems(ctx, repository.pool, transaction.ID)
		if err != nil {
			return service.ReportPage{}, err
		}
		transaction.History, err = repository.loadHistory(ctx, repository.pool, transaction.ID)
		if err != nil {
			return service.ReportPage{}, err
		}

		reports = append(reports, transaction)
	}

	if err := rows.Err(); err != nil {
		return service.ReportPage{}, fmt.Errorf("iterate reports: %w", err)
	}

	return service.ReportPage{
		Items:    reports,
		Total:    total,
		Limit:    filters.Limit,
		Offset:   filters.Offset,
		UnitsIn:  unitsIn,
		UnitsOut: unitsOut,
	}, nil
}

func (repository *Repository) ListAllReports(ctx context.Context, filters service.ReportFilters) ([]domain.Transaction, error) {
	_, completedFrom, completedTo, referenceCode, typeFilter := reportQueryArgs(filters)

	rows, err := repository.pool.Query(ctx, `
		SELECT id, type, status, reference_code, note, completed_at, created_at, updated_at
		FROM stock_transactions
		WHERE status = $1
		  AND type IN ($2, $3)
		  AND ($4 = '' OR type = $4)
		  AND ($5 = '' OR reference_code ILIKE '%' || $5 || '%')
		  AND ($6::timestamptz IS NULL OR completed_at >= $6::timestamptz)
		  AND ($7::timestamptz IS NULL OR completed_at < $7::timestamptz)
		ORDER BY completed_at DESC NULLS LAST, created_at DESC, id DESC
	`, domain.StatusDone, domain.TypeStockIn, domain.TypeStockOut, typeFilter, referenceCode, completedFrom, completedTo)
	if err != nil {
		return nil, fmt.Errorf("list all reports: %w", err)
	}
	defer rows.Close()

	reports := make([]domain.Transaction, 0)
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}

		transaction.Items, err = repository.loadTransactionItems(ctx, repository.pool, transaction.ID)
		if err != nil {
			return nil, err
		}
		transaction.History, err = repository.loadHistory(ctx, repository.pool, transaction.ID)
		if err != nil {
			return nil, err
		}

		reports = append(reports, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all reports: %w", err)
	}

	return reports, nil
}

func reportQueryArgs(filters service.ReportFilters) ([]any, any, any, string, string) {
	typeFilter := ""
	if filters.Type != nil {
		typeFilter = string(*filters.Type)
	}

	referenceCode := filters.ReferenceCode

	var completedFrom any
	if filters.CompletedFrom != nil {
		completedFrom = *filters.CompletedFrom
	}

	var completedTo any
	if filters.CompletedTo != nil {
		completedTo = *filters.CompletedTo
	}

	return []any{domain.StatusDone, domain.TypeStockIn, domain.TypeStockOut, typeFilter, referenceCode, completedFrom, completedTo}, completedFrom, completedTo, referenceCode, typeFilter
}

func (repository *Repository) insertTransaction(ctx context.Context, tx pgx.Tx, transactionType domain.TransactionType, status domain.TransactionStatus, referenceCode, note string) (domain.Transaction, error) {
	var transaction domain.Transaction
	if err := tx.QueryRow(ctx, `
		INSERT INTO stock_transactions (type, status, reference_code, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, type, status, reference_code, note, completed_at, created_at, updated_at
	`, transactionType, status, referenceCode, note).Scan(
		&transaction.ID,
		&transaction.Type,
		&transaction.Status,
		&transaction.ReferenceCode,
		&transaction.Note,
		&transaction.CompletedAt,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	); err != nil {
		if uniqueViolation(err) {
			return domain.Transaction{}, service.Conflict("reference code already exists")
		}

		return domain.Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}

	return transaction, nil
}

func (repository *Repository) insertItemsFromInput(ctx context.Context, tx pgx.Tx, transactionID int64, items []service.TransactionItemInput) error {
	for _, requestItem := range items {
		inventory, err := repository.lockInventory(ctx, tx, requestItem.InventoryID)
		if err != nil {
			return err
		}
		if err := repository.insertTransactionItem(ctx, tx, transactionID, domain.TransactionItem{
			InventoryID:  inventory.ID,
			SKU:          inventory.SKU,
			Name:         inventory.Name,
			CustomerName: inventory.CustomerName,
			Quantity:     requestItem.Quantity,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (repository *Repository) insertTransactionItem(ctx context.Context, tx pgx.Tx, transactionID int64, item domain.TransactionItem) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO stock_transaction_items (transaction_id, inventory_item_id, sku, item_name, customer_name, quantity)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, transactionID, item.InventoryID, item.SKU, item.Name, item.CustomerName, item.Quantity); err != nil {
		return fmt.Errorf("insert transaction item: %w", err)
	}

	return nil
}

func (repository *Repository) insertReservation(ctx context.Context, tx pgx.Tx, transactionID, inventoryID, quantity int64) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO stock_reservations (transaction_id, inventory_item_id, quantity, status)
		VALUES ($1, $2, $3, $4)
	`, transactionID, inventoryID, quantity, domain.ReservationActive); err != nil {
		return fmt.Errorf("insert reservation: %w", err)
	}

	return nil
}

func (repository *Repository) insertHistory(ctx context.Context, tx pgx.Tx, transactionID int64, status domain.TransactionStatus, note string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO stock_transaction_history (transaction_id, status, note)
		VALUES ($1, $2, $3)
	`, transactionID, status, note); err != nil {
		return fmt.Errorf("insert transaction history: %w", err)
	}

	return nil
}

func (repository *Repository) updateTransactionStatus(ctx context.Context, tx pgx.Tx, transactionID int64, status domain.TransactionStatus, note string) error {
	if _, err := tx.Exec(ctx, `
		UPDATE stock_transactions
		SET status = $2,
		    note = CASE WHEN $3 = '' THEN note ELSE $3 END,
		    completed_at = CASE WHEN $2 = $4 THEN NOW() ELSE completed_at END,
		    updated_at = NOW()
		WHERE id = $1
	`, transactionID, status, note, domain.StatusDone); err != nil {
		return fmt.Errorf("update transaction status: %w", err)
	}

	return nil
}

func (repository *Repository) lockInventory(ctx context.Context, tx pgx.Tx, inventoryID int64) (domain.Inventory, error) {
	var inventory domain.Inventory
	if err := tx.QueryRow(ctx, `
		SELECT id, sku, name, customer_name, physical_stock, reserved_stock, created_at, updated_at
		FROM inventory_items
		WHERE id = $1
		FOR UPDATE
	`, inventoryID).Scan(
		&inventory.ID,
		&inventory.SKU,
		&inventory.Name,
		&inventory.CustomerName,
		&inventory.PhysicalStock,
		&inventory.ReservedStock,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Inventory{}, service.NotFound("inventory item not found")
		}

		return domain.Inventory{}, fmt.Errorf("lock inventory: %w", err)
	}
	inventory.RefreshAvailable()

	return inventory, nil
}

func (repository *Repository) lockTransaction(ctx context.Context, tx pgx.Tx, transactionID int64) (domain.Transaction, error) {
	row := tx.QueryRow(ctx, `
		SELECT id, type, status, reference_code, note, completed_at, created_at, updated_at
		FROM stock_transactions
		WHERE id = $1
		FOR UPDATE
	`, transactionID)

	transaction, err := scanTransaction(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, service.ErrNotFound) {
			return domain.Transaction{}, service.NotFound("transaction not found")
		}

		return domain.Transaction{}, err
	}

	return transaction, nil
}

func (repository *Repository) loadTransaction(ctx context.Context, db queryable, transactionID int64) (domain.Transaction, error) {
	row := db.QueryRow(ctx, `
		SELECT id, type, status, reference_code, note, completed_at, created_at, updated_at
		FROM stock_transactions
		WHERE id = $1
	`, transactionID)

	transaction, err := scanTransaction(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Transaction{}, service.NotFound("transaction not found")
		}

		return domain.Transaction{}, err
	}

	transaction.Items, err = repository.loadTransactionItems(ctx, db, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	transaction.History, err = repository.loadHistory(ctx, db, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}

	return transaction, nil
}

func (repository *Repository) loadTransactionItems(ctx context.Context, db queryable, transactionID int64) ([]domain.TransactionItem, error) {
	rows, err := db.Query(ctx, `
		SELECT inventory_item_id, sku, item_name, customer_name, quantity
		FROM stock_transaction_items
		WHERE transaction_id = $1
		ORDER BY id ASC
	`, transactionID)
	if err != nil {
		return nil, fmt.Errorf("query transaction items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.TransactionItem, 0)
	for rows.Next() {
		var item domain.TransactionItem
		if err := rows.Scan(&item.InventoryID, &item.SKU, &item.Name, &item.CustomerName, &item.Quantity); err != nil {
			return nil, fmt.Errorf("scan transaction item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transaction items: %w", err)
	}

	return items, nil
}

func (repository *Repository) loadHistory(ctx context.Context, db queryable, transactionID int64) ([]domain.HistoryEntry, error) {
	rows, err := db.Query(ctx, `
		SELECT status, note, created_at
		FROM stock_transaction_history
		WHERE transaction_id = $1
		ORDER BY id ASC
	`, transactionID)
	if err != nil {
		return nil, fmt.Errorf("query transaction history: %w", err)
	}
	defer rows.Close()

	history := make([]domain.HistoryEntry, 0)
	for rows.Next() {
		var entry domain.HistoryEntry
		if err := rows.Scan(&entry.Status, &entry.Note, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan transaction history: %w", err)
		}
		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transaction history: %w", err)
	}

	return history, nil
}

type reservation struct {
	InventoryID int64
	Quantity    int64
}

func (repository *Repository) loadReservations(ctx context.Context, db queryable, transactionID int64, status domain.ReservationStatus) ([]reservation, error) {
	rows, err := db.Query(ctx, `
		SELECT inventory_item_id, quantity
		FROM stock_reservations
		WHERE transaction_id = $1 AND status = $2
		ORDER BY id ASC
	`, transactionID, status)
	if err != nil {
		return nil, fmt.Errorf("query reservations: %w", err)
	}
	defer rows.Close()

	reservations := make([]reservation, 0)
	for rows.Next() {
		var item reservation
		if err := rows.Scan(&item.InventoryID, &item.Quantity); err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		reservations = append(reservations, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reservations: %w", err)
	}

	return reservations, nil
}

func scanTransaction(row interface{ Scan(...any) error }) (domain.Transaction, error) {
	var transaction domain.Transaction
	if err := row.Scan(
		&transaction.ID,
		&transaction.Type,
		&transaction.Status,
		&transaction.ReferenceCode,
		&transaction.Note,
		&transaction.CompletedAt,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	); err != nil {
		return domain.Transaction{}, err
	}

	return transaction, nil
}

func uniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code == "23505"
	}

	return false
}
