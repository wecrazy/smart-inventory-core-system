# Manual Testing Guide

This guide describes the operator steps to validate the full workflow from the frontend through the final report, matching the assessment requirements.

## 1. Start The Application

From the repository root:

```bash
make env
make install
make schema
make dev
```

Then open:

- Frontend: `http://localhost:5173`
- Swagger UI: `http://localhost:8080/swagger/index.html`

## 2. Prepare Inventory Data

Open the `Inventory` page.

Create at least these two items so stock-in, stock-out, and reports can all be tested:

### Item 1

- SKU: `SKU-001`
- Name: `Widget A`
- Customer: `Acme Corp`
- Initial physical stock: `100`

### Item 2

- SKU: `SKU-002`
- Name: `Widget B`
- Customer: `Beta Retail`
- Initial physical stock: `40`

Expected result:

- the inventory table shows the newly created items
- `physical stock` matches the initial value
- `reserved stock` is still `0`
- `available stock` matches `physical stock`

If the table looks empty because filters are active, clear the search, SKU, and customer fields.

## 3. Test Stock Adjustment

Still on the `Inventory` page, use the `Stock adjustment` panel.

Suggested test:

- choose `SKU-001`
- set `New physical stock` to `120`
- set `Reference code` to `ADJ-MANUAL-001`
- set note to `Cycle count correction`

Expected result:

- a success message appears in the frontend
- the item `physical stock` changes to `120`
- `available stock` changes to `120`
- the change is stored as an auditable adjustment transaction in the backend, but it does not appear in the report page because reports are limited to `DONE` stock-in and stock-out transactions

## 4. Test The Stock In Flow

Open the `Stock In` page.

Create a new transaction:

- reference code: `IN-MANUAL-001`
- note: `Morning receipt`
- item: `SKU-001`
- quantity: `10`

After saving, the transaction should appear with status `CREATED`.

Continue the status flow:

1. Click `Move to in progress`
2. Confirm the status becomes `IN_PROGRESS`
3. Go back to the `Inventory` page
4. Confirm `physical stock` has not increased yet
5. Return to `Stock In`
6. Click `Mark done`

Expected result:

- the status becomes `DONE`
- the item `physical stock` increases by `10`
- `available stock` also increases by `10`
- the transaction now appears on the `Reports` page

## 5. Test The Two-Phase Stock Out Flow

Open the `Stock Out` page.

Create a new transaction:

- reference code: `OUT-MANUAL-001`
- note: `Shipment batch 1`
- item: `SKU-001`
- quantity: `15`

Expected allocation result:

- the transaction is created with status `ALLOCATED`
- on the `Inventory` page, `reserved stock` increases by `15`
- `available stock` decreases by `15`
- `physical stock` does not change yet

Now validate both scenarios below.

### Scenario A: Cancel And Roll Back

1. On transaction `OUT-MANUAL-001`, click `Move to in progress`
2. Confirm the status becomes `IN_PROGRESS`
3. Click `Cancel and rollback`

Expected result:

- the status becomes `CANCELLED`
- `reserved stock` goes back down
- `available stock` returns to the pre-allocation value
- `physical stock` remains unchanged
- the transaction does not appear on the `Reports` page

### Scenario B: Finish To DONE

Create another transaction:

- reference code: `OUT-MANUAL-002`
- note: `Final shipment`
- item: `SKU-001`
- quantity: `20`

Then:

1. Click `Move to in progress`
2. Click `Mark done`

Expected result:

- the status becomes `DONE`
- `physical stock` decreases by `20`
- `reserved stock` is released for the completed quantity
- the transaction appears on the `Reports` page

## 6. Test Reports

Open the `Reports` page.

This page must only display:

- `STOCK_IN` transactions with status `DONE`
- `STOCK_OUT` transactions with status `DONE`

This page must not display:

- adjustments
- stock-in transactions still in `CREATED`, `IN_PROGRESS`, or `CANCELLED`
- stock-out transactions still in `ALLOCATED`, `IN_PROGRESS`, or `CANCELLED`

If there is no report data yet, the frontend now shows an explicit empty state explaining that at least one stock-in or stock-out transaction must reach `DONE` first.

When report data exists, also verify these behaviors:

- the reports page now loads `10` completed transactions per page from the server instead of fetching the entire report set at once
- the operator can filter reports by transaction type, partial reference code, and completed date range
- the report summary now separates `Units moved in` and `Units moved out`
- each transaction report card starts collapsed, and the operator can click `Show details` to expand item detail and status history
- when `Print report` is clicked, the print output now focuses on the report section rather than the full application shell

When report data exists, the operator can also:

- click `Print report` to open the browser print dialog with a print-friendly report layout
- click `Export CSV` to download every report that matches the active filters from the server as a CSV file

## 7. Requirement Checklist

Quick validation checklist:

- [ ] inventory can be searched by name, SKU, and customer
- [ ] inventory separates `physical`, `reserved`, and `available`
- [ ] stock adjustment is available from the UI
- [ ] stock-in follows `CREATED -> IN_PROGRESS -> DONE`
- [ ] stock-in can be cancelled before `DONE`
- [ ] stock-out follows `ALLOCATED -> IN_PROGRESS -> DONE`
- [ ] cancelling stock-out releases the reservation
- [ ] reports only show `DONE` transactions
- [ ] detailed transaction reports can be printed and exported as CSV
- [ ] reports are loaded incrementally from the server and expanded through an accordion-style detail view
- [ ] reports can be filtered by type, reference code, and completed date range
- [ ] the frontend now shows meaningful empty states and mutation feedback for operators
