CREATE TABLE IF NOT EXISTS inventory_items (
	id BIGSERIAL PRIMARY KEY,
	sku TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	customer_name TEXT NOT NULL,
	physical_stock BIGINT NOT NULL DEFAULT 0 CHECK (physical_stock >= 0),
	reserved_stock BIGINT NOT NULL DEFAULT 0 CHECK (reserved_stock >= 0),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CHECK (physical_stock >= reserved_stock)
);

CREATE INDEX IF NOT EXISTS idx_inventory_items_name ON inventory_items (name);
CREATE INDEX IF NOT EXISTS idx_inventory_items_customer_name ON inventory_items (customer_name);

CREATE TABLE IF NOT EXISTS stock_transactions (
	id BIGSERIAL PRIMARY KEY,
	type TEXT NOT NULL CHECK (type IN ('STOCK_IN', 'STOCK_OUT', 'ADJUSTMENT')),
	status TEXT NOT NULL CHECK (status IN ('CREATED', 'ALLOCATED', 'IN_PROGRESS', 'DONE', 'CANCELLED')),
	reference_code TEXT NOT NULL UNIQUE,
	note TEXT NOT NULL DEFAULT '',
	completed_at TIMESTAMPTZ NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_transactions_type_status ON stock_transactions (type, status, created_at DESC);

CREATE TABLE IF NOT EXISTS stock_transaction_items (
	id BIGSERIAL PRIMARY KEY,
	transaction_id BIGINT NOT NULL REFERENCES stock_transactions(id) ON DELETE CASCADE,
	inventory_item_id BIGINT NOT NULL REFERENCES inventory_items(id),
	sku TEXT NOT NULL,
	item_name TEXT NOT NULL,
	customer_name TEXT NOT NULL,
	quantity BIGINT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_transaction_items_transaction_id ON stock_transaction_items (transaction_id);
CREATE INDEX IF NOT EXISTS idx_stock_transaction_items_inventory_item_id ON stock_transaction_items (inventory_item_id);

CREATE TABLE IF NOT EXISTS stock_transaction_history (
	id BIGSERIAL PRIMARY KEY,
	transaction_id BIGINT NOT NULL REFERENCES stock_transactions(id) ON DELETE CASCADE,
	status TEXT NOT NULL CHECK (status IN ('CREATED', 'ALLOCATED', 'IN_PROGRESS', 'DONE', 'CANCELLED')),
	note TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_transaction_history_transaction_id ON stock_transaction_history (transaction_id);

CREATE TABLE IF NOT EXISTS stock_reservations (
	id BIGSERIAL PRIMARY KEY,
	transaction_id BIGINT NOT NULL REFERENCES stock_transactions(id) ON DELETE CASCADE,
	inventory_item_id BIGINT NOT NULL REFERENCES inventory_items(id),
	quantity BIGINT NOT NULL CHECK (quantity > 0),
	status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'FULFILLED', 'RELEASED')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_reservations_transaction_status ON stock_reservations (transaction_id, status);
CREATE INDEX IF NOT EXISTS idx_stock_reservations_inventory_status ON stock_reservations (inventory_item_id, status);