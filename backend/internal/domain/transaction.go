package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidStatus     = errors.New("invalid status")
	ErrInvalidType       = errors.New("invalid transaction type")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type TransactionType string

const (
	TypeStockIn    TransactionType = "STOCK_IN"
	TypeStockOut   TransactionType = "STOCK_OUT"
	TypeAdjustment TransactionType = "ADJUSTMENT"
)

type TransactionStatus string

const (
	StatusCreated    TransactionStatus = "CREATED"
	StatusAllocated  TransactionStatus = "ALLOCATED"
	StatusInProgress TransactionStatus = "IN_PROGRESS"
	StatusDone       TransactionStatus = "DONE"
	StatusCancelled  TransactionStatus = "CANCELLED"
)

type ReservationStatus string

const (
	ReservationActive    ReservationStatus = "ACTIVE"
	ReservationFulfilled ReservationStatus = "FULFILLED"
	ReservationReleased  ReservationStatus = "RELEASED"
)

type Transaction struct {
	ID            int64             `json:"id" example:"17"`
	Type          TransactionType   `json:"type" example:"STOCK_IN" enums:"STOCK_IN,STOCK_OUT,ADJUSTMENT"`
	Status        TransactionStatus `json:"status" example:"DONE" enums:"CREATED,ALLOCATED,IN_PROGRESS,DONE,CANCELLED"`
	ReferenceCode string            `json:"referenceCode" example:"IN-20260401-001"`
	Note          string            `json:"note" example:"Morning warehouse receipt"`
	CompletedAt   *time.Time        `json:"completedAt,omitempty" example:"2026-04-01T12:15:00Z"`
	CreatedAt     time.Time         `json:"createdAt" example:"2026-04-01T09:00:00Z"`
	UpdatedAt     time.Time         `json:"updatedAt" example:"2026-04-01T12:15:00Z"`
	Items         []TransactionItem `json:"items"`
	History       []HistoryEntry    `json:"history"`
}

type TransactionItem struct {
	InventoryID  int64  `json:"inventoryId" example:"1"`
	SKU          string `json:"sku" example:"SKU-001"`
	Name         string `json:"name" example:"Widget A"`
	CustomerName string `json:"customerName" example:"Acme Corp"`
	Quantity     int64  `json:"quantity" example:"10"`
}

type HistoryEntry struct {
	Status    TransactionStatus `json:"status" example:"IN_PROGRESS" enums:"CREATED,ALLOCATED,IN_PROGRESS,DONE,CANCELLED"`
	Note      string            `json:"note" example:"Checked and ready for next step"`
	CreatedAt time.Time         `json:"createdAt" example:"2026-04-01T10:30:00Z"`
}

func ParseTransactionType(value string) (TransactionType, error) {
	transactionType := TransactionType(strings.ToUpper(strings.TrimSpace(value)))

	switch transactionType {
	case TypeStockIn, TypeStockOut, TypeAdjustment:
		return transactionType, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidType, value)
	}
}

func ParseTransactionStatus(value string) (TransactionStatus, error) {
	status := TransactionStatus(strings.ToUpper(strings.TrimSpace(value)))

	switch status {
	case StatusCreated, StatusAllocated, StatusInProgress, StatusDone, StatusCancelled:
		return status, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidStatus, value)
	}
}

func (transactionType TransactionType) ReferencePrefix() string {
	switch transactionType {
	case TypeStockIn:
		return "IN"
	case TypeStockOut:
		return "OUT"
	case TypeAdjustment:
		return "ADJ"
	default:
		return "TX"
	}
}

func (transaction Transaction) IsReportable() bool {
	return transaction.Status == StatusDone && (transaction.Type == TypeStockIn || transaction.Type == TypeStockOut)
}

func ValidateStockInTransition(current, next TransactionStatus) error {
	switch current {
	case StatusCreated:
		if next == StatusInProgress || next == StatusCancelled {
			return nil
		}
	case StatusInProgress:
		if next == StatusDone || next == StatusCancelled {
			return nil
		}
	}

	return fmt.Errorf("%w: stock-in %s -> %s", ErrInvalidTransition, current, next)
}

func ValidateStockOutTransition(current, next TransactionStatus) error {
	switch current {
	case StatusAllocated:
		if next == StatusInProgress || next == StatusCancelled {
			return nil
		}
	case StatusInProgress:
		if next == StatusDone || next == StatusCancelled {
			return nil
		}
	}

	return fmt.Errorf("%w: stock-out %s -> %s", ErrInvalidTransition, current, next)
}
