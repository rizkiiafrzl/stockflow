-- Migration: 001_init_schema.sql
-- Applied automatically via docker-entrypoint-initdb.d/

-- Products table
CREATE TABLE IF NOT EXISTS products (
    id         SERIAL PRIMARY KEY,
    sku        TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    price      INTEGER NOT NULL CHECK (price >= 0),
    stock      INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Stock movements table
CREATE TABLE IF NOT EXISTS stock_movements (
    id         SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id),
    type       TEXT NOT NULL CHECK (type IN ('in', 'out')),
    qty        INTEGER NOT NULL CHECK (qty > 0),
    note       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sales orders table
CREATE TABLE IF NOT EXISTS sales_orders (
    id           SERIAL PRIMARY KEY,
    customer     TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'CREATED' CHECK (status IN ('CREATED','CANCELLED')),
    total        INTEGER NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    cancelled_at TIMESTAMPTZ
);

-- Sales order items table
CREATE TABLE IF NOT EXISTS sales_order_items (
    id         SERIAL PRIMARY KEY,
    order_id   INTEGER NOT NULL REFERENCES sales_orders(id),
    sku        TEXT NOT NULL REFERENCES products(sku),
    qty        INTEGER NOT NULL CHECK (qty > 0),
    unit_price INTEGER NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_movements_product_id ON stock_movements(product_id);
CREATE INDEX IF NOT EXISTS idx_movements_created_at ON stock_movements(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status ON sales_orders(status);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON sales_order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
