-- Seed data: 12 realistic products + initial stock + sample orders
-- Applied automatically after 001_init_schema.sql

-- Insert 12 realistic products
INSERT INTO products (sku, name, price, stock, is_active) VALUES
  ('SKU-001', 'Laptop ASUS VivoBook 15',       8500000,  45, TRUE),
  ('SKU-002', 'Mouse Wireless Logitech M235',    250000, 120, TRUE),
  ('SKU-003', 'Keyboard Mechanical Rexus K9',    650000,  60, TRUE),
  ('SKU-004', 'Monitor LG 24 inch FHD',         2800000,  18, TRUE),
  ('SKU-005', 'Headset Gaming HyperX Cloud',    1200000,  35, TRUE),
  ('SKU-006', 'SSD Samsung 500GB SATA',          950000,  80, TRUE),
  ('SKU-007', 'RAM DDR4 8GB Corsair',            450000,  90, TRUE),
  ('SKU-008', 'Webcam Logitech C270',            450000,  25, TRUE),
  ('SKU-009', 'USB Hub 7-Port Orico',            180000,  55, TRUE),
  ('SKU-010', 'Mousepad Gaming XL 80x30cm',       85000,   8, TRUE),
  ('SKU-011', 'Cooling Pad Laptop Deepcool',     220000,  12, TRUE),
  ('SKU-012', 'HDMI Cable 2m Ugreen',             65000,   5, FALSE)
ON CONFLICT (sku) DO NOTHING;

-- Initial stock-in movements (simulate receiving inventory)
INSERT INTO stock_movements (product_id, type, qty, note)
SELECT id, 'in', stock, 'Initial inventory' FROM products
WHERE stock > 0 AND NOT EXISTS (SELECT 1 FROM stock_movements sm WHERE sm.product_id = products.id);

-- Sample order 1: office supplies order
WITH o AS (
  INSERT INTO sales_orders (customer, total)
  VALUES ('PT Maju Bersama', 10650000)
  RETURNING id
),
i1 AS (
  INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
  SELECT o.id, 'SKU-001', 1, 8500000 FROM o
),
i2 AS (
  INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
  SELECT o.id, 'SKU-002', 2, 250000 FROM o
),
i3 AS (
  INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
  SELECT o.id, 'SKU-009', 1, 180000 FROM o
),
upd1 AS (UPDATE products SET stock = stock - 1 WHERE sku = 'SKU-001'),
upd2 AS (UPDATE products SET stock = stock - 2 WHERE sku = 'SKU-002'),
upd3 AS (UPDATE products SET stock = stock - 1 WHERE sku = 'SKU-009'),
mv1 AS (INSERT INTO stock_movements (product_id, type, qty, note) SELECT id, 'out', 1, 'Sale order #1' FROM products WHERE sku = 'SKU-001'),
mv2 AS (INSERT INTO stock_movements (product_id, type, qty, note) SELECT id, 'out', 2, 'Sale order #1' FROM products WHERE sku = 'SKU-002'),
mv3 AS (INSERT INTO stock_movements (product_id, type, qty, note) SELECT id, 'out', 1, 'Sale order #1' FROM products WHERE sku = 'SKU-009')
SELECT 1;

-- Sample order 2: gaming setup order
WITH o AS (
  INSERT INTO sales_orders (customer, total)
  VALUES ('Budi Santoso', 2450000)
  RETURNING id
),
i1 AS (
  INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
  SELECT o.id, 'SKU-005', 1, 1200000 FROM o
),
i2 AS (
  INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
  SELECT o.id, 'SKU-003', 1, 650000 FROM o
),
i3 AS (
  INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
  SELECT o.id, 'SKU-010', 3, 85000 FROM o
),
i4 AS (
  INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
  SELECT o.id, 'SKU-009', 1, 180000 FROM o
),
upd1 AS (UPDATE products SET stock = stock - 1 WHERE sku = 'SKU-005'),
upd2 AS (UPDATE products SET stock = stock - 1 WHERE sku = 'SKU-003'),
upd3 AS (UPDATE products SET stock = stock - 3 WHERE sku = 'SKU-010'),
upd4 AS (UPDATE products SET stock = stock - 1 WHERE sku = 'SKU-009'),
mv1 AS (INSERT INTO stock_movements (product_id, type, qty, note) SELECT id, 'out', 1, 'Sale order #2' FROM products WHERE sku = 'SKU-005'),
mv2 AS (INSERT INTO stock_movements (product_id, type, qty, note) SELECT id, 'out', 1, 'Sale order #2' FROM products WHERE sku = 'SKU-003'),
mv3 AS (INSERT INTO stock_movements (product_id, type, qty, note) SELECT id, 'out', 3, 'Sale order #2' FROM products WHERE sku = 'SKU-010'),
mv4 AS (INSERT INTO stock_movements (product_id, type, qty, note) SELECT id, 'out', 1, 'Sale order #2' FROM products WHERE sku = 'SKU-009')
SELECT 1;
