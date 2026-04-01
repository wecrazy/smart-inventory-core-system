package domain

import "testing"

func TestValidateStockInTransition(t *testing.T) {
	tests := []struct {
		name    string
		current TransactionStatus
		next    TransactionStatus
		wantErr bool
	}{
		{name: "created to in progress", current: StatusCreated, next: StatusInProgress},
		{name: "created to cancelled", current: StatusCreated, next: StatusCancelled},
		{name: "in progress to done", current: StatusInProgress, next: StatusDone},
		{name: "in progress to cancelled", current: StatusInProgress, next: StatusCancelled},
		{name: "created to done invalid", current: StatusCreated, next: StatusDone, wantErr: true},
		{name: "done to cancelled invalid", current: StatusDone, next: StatusCancelled, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateStockInTransition(test.current, test.next)
			if test.wantErr && err == nil {
				t.Fatalf("expected error for %s -> %s", test.current, test.next)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected no error for %s -> %s, got %v", test.current, test.next, err)
			}
		})
	}
}

func TestValidateStockOutTransition(t *testing.T) {
	tests := []struct {
		name    string
		current TransactionStatus
		next    TransactionStatus
		wantErr bool
	}{
		{name: "allocated to in progress", current: StatusAllocated, next: StatusInProgress},
		{name: "allocated to cancelled", current: StatusAllocated, next: StatusCancelled},
		{name: "in progress to done", current: StatusInProgress, next: StatusDone},
		{name: "in progress to cancelled", current: StatusInProgress, next: StatusCancelled},
		{name: "allocated to done invalid", current: StatusAllocated, next: StatusDone, wantErr: true},
		{name: "done to cancelled invalid", current: StatusDone, next: StatusCancelled, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateStockOutTransition(test.current, test.next)
			if test.wantErr && err == nil {
				t.Fatalf("expected error for %s -> %s", test.current, test.next)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("expected no error for %s -> %s, got %v", test.current, test.next, err)
			}
		})
	}
}

func TestParseTransactionStatus(t *testing.T) {
	status, err := ParseTransactionStatus("in_progress")
	if err != nil {
		t.Fatalf("expected valid status parse, got %v", err)
	}
	if status != StatusInProgress {
		t.Fatalf("expected %s, got %s", StatusInProgress, status)
	}

	if _, err := ParseTransactionStatus("unknown"); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestReferencePrefix(t *testing.T) {
	tests := map[TransactionType]string{
		TypeStockIn:    "IN",
		TypeStockOut:   "OUT",
		TypeAdjustment: "ADJ",
	}

	for transactionType, expected := range tests {
		if got := transactionType.ReferencePrefix(); got != expected {
			t.Fatalf("expected prefix %s for %s, got %s", expected, transactionType, got)
		}
	}
}
