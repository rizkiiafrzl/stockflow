# Stockflow

A full-stack warehouse management application built for warehouse supervisors to manage products, adjust stock levels, and record sales orders that safely consume inventory under concurrent load.

Built as an end-to-end vertical slice: **PostgreSQL → Go REST API → React + TypeScript (Vite)**.

---

## Key Features & Architectural Highlights

1. **Concurrency Safety & Atomic Inventory**:
   - Guarded against race conditions and overselling using atomic SQL conditional updates (`UPDATE products SET stock = stock - $qty WHERE id = $id AND stock >= $qty`) and pessimistic locking (`SELECT ... FOR UPDATE`) within ACID transactions.
   - Zero negative stock guarantee even with high concurrent order surges.
2. **Cross-Stack Error Contract**:
   - Structured error envelope `{ "error": { "code": "...", "sku": "...", "available": ... } }`.
   - Backend `409 Conflict` (`INSUFFICIENT_STOCK`, `ALREADY_CANCELLED`, `PRODUCT_HAS_HISTORY`) propagates directly to user-facing UI highlights with actionable error banners.
3. **Derived Real-Time Metadata (`last_movement_at`)**:
   - Calculated via `MAX(sm.created_at)` from `stock_movements` in SQL queries, avoiding denormalization drifts or cache staleness while maintaining relational integrity.
4. **Order Lifecycle & Stock Restoration**:
   - Multi-item orders validated atomically before any record insertion.
   - Order cancellation restores exact stock levels and writes corresponding `in` movements within a single atomic transaction.

---

## Repository Layout

```
stockflow/
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go              # Service entry point & router setup
│   ├── internal/
│   │   ├── handler/                 # HTTP layer (chi routes, JSON codecs)
│   │   │   ├── helpers.go
│   │   │   ├── orders.go
│   │   │   └── products.go
│   │   ├── model/                   # Domain entities and DTO contracts
│   │   │   └── types.go
│   │   └── repository/              # Persistence & transactional logic
│   │       ├── order_repo.go
│   │       ├── order_repo_test.go   # Concurrency & stock rollback tests
│   │       ├── product_repo.go
│   │       ├── product_repo_test.go # Stock adjustments & delete constraints
│   │       └── setup_test.go        # Test DB runner & schema initializer
│   ├── migrations/
│   │   ├── 001_init_schema.sql      # Schema with foreign keys & indexes
│   │   └── 002_seed.sql             # 12 initial products & sample orders
│   ├── Dockerfile                   # Multi-stage Go alpine container
│   ├── .env                         # Local environment configuration
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── api/
│   │   │   └── client.ts            # Centralized typed HTTP client
│   │   ├── components/
│   │   │   ├── CreateOrderModal.tsx # Sales order form with live SKU validation
│   │   │   ├── ProductModal.tsx     # Product create & edit modal
│   │   │   ├── StockModal.tsx       # Stock in / out adjustment with 409 handling
│   │   │   └── SummaryCards.tsx     # Metrics dashboard (total products, low stock, etc.)
│   │   ├── pages/
│   │   │   ├── OrderDetailPage.tsx # Order line items & cancellation
│   │   │   └── ProductsPage.tsx     # Paginated table, search & actions
│   │   ├── types/
│   │   │   └── index.ts             # TypeScript definitions matching API DTOs
│   │   ├── utils/
│   │   │   ├── format.ts            # Currency (IDR) & date formatting
│   │   │   ├── validation.ts        # Pure form validation logic
│   │   │   └── validation.test.ts   # Vitest unit tests for validation rules
│   │   ├── App.tsx                  # Root navigation & layout
│   │   ├── index.css                # Polished design system (dark theme)
│   │   └── main.tsx
│   ├── Dockerfile                   # Multi-stage Vite + Nginx container
│   ├── nginx.conf                   # Reverse proxy for /api and SPA fallback
│   ├── package.json
│   └── vite.config.ts               # Local dev proxy configuration
├── docker-compose.yml               # Container orchestration (DB, API, UI)
└── README.md
```

---

## How to Run

### Option 1: Docker Compose (All-in-One)

Prerequisite: Docker & Docker Compose installed.

```bash
# Clone the repository and run:
docker compose up --build -d
```

- **Frontend UI**: http://localhost:5173
- **Backend API**: http://localhost:8080
- **PostgreSQL**: `localhost:5432` (user: `postgres`, password: `1234`, db: `stockflow_db`)

The PostgreSQL container automatically runs migrations and seeds initial products on first launch via `/docker-entrypoint-initdb.d`.

---

### Option 2: Running Locally (Development Mode)

#### 1. Database (PostgreSQL)
Ensure PostgreSQL is running on `localhost:5432` with a database named `stockflow_db`.
Copy the example environment file and adjust credentials if needed:
```bash
cd backend
cp .env.example .env    # Linux / macOS
copy .env.example .env  # Windows
```
Default configuration (`backend/.env.example`):
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=1234
DB_NAME=stockflow_db
```
*(Note: Sensible defaults are also embedded as code-level fallbacks in the Go service).*

#### 2. Backend Service
```bash
cd backend
go run ./cmd/api
```
*Note: The backend automatically checks and executes schema migrations and seeds on startup if the tables do not exist.*

#### 3. Frontend Web App
```bash
cd frontend
npm install
npm run dev
```
Open http://localhost:5173 in your browser. Requests to `/api/*` are automatically proxied to the backend on `http://localhost:8080`.

---

## Running Tests

### Backend Tests (Go)
Includes concurrency race testing, stock overdraw validation, rollback guarantees, and deletion safety:
```bash
cd backend
go test -v ./internal/repository/...
```

To run with the race detector enabled (on supported environments with CGO):
```bash
go test -race -v ./internal/repository/...
```

### Frontend Tests (Vitest)
Includes unit tests verifying form validation (positive quantities, required SKUs, non-negative integer pricing, and duplicate SKU prevention):
```bash
cd frontend
npm test
```

---

## Design Decisions & Trade-offs

### 1. Concurrency Control & Stock Safety
- **Anti-pattern avoided**: `SELECT stock FROM products ...` followed by application-level checks and `UPDATE products SET stock = ...`. Under concurrent orders, multiple requests read the same initial stock and both proceed, resulting in negative stock.
- **Solution implemented**: Atomic update queries with conditional predicates:
  ```sql
  UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1
  ```
  Combined with `SELECT ... FOR UPDATE` row locks inside single transactions, this ensures serializability. If rows affected is 0 or stock is insufficient, the transaction cleanly rolls back and returns HTTP `409 Conflict` with `INSUFFICIENT_STOCK`.

### 2. Derived `last_movement_at`
- **Choice**: Derived dynamically using `MAX(sm.created_at)` from `stock_movements` via `LEFT JOIN` and `GROUP BY`.
- **Rationale**: Storing `last_movement_at` as a column on `products` requires dual-writes or database triggers, which can become out of sync or create lock contention. Because `stock_movements` has an index on `product_id`, the aggregated query remains fast while guaranteeing 100% relational accuracy.

### 3. Currency Representation
- All monetary amounts (prices and order totals) are stored as 64-bit integer IDR (Rupiah), completely avoiding floating-point rounding errors.

### 4. Cross-Stack Error Handling
- Structured API response on errors:
  ```json
  {
    "error": {
      "code": "INSUFFICIENT_STOCK",
      "sku": "SKU-001",
      "available": 2
    }
  }
  ```
- The frontend UI specifically catches this error code, highlights the problematic SKU input in red, and renders a banner displaying the exact available quantity.

---

## API Contract & Ready-to-Run cURL Examples

Assume the backend server is listening on `http://localhost:8080`.

### 1. List Products (with search, pagination, and derived `last_movement_at`)
```bash
curl -X GET "http://localhost:8080/api/products?search=asus&page=1&limit=10"
```

### 2. Create Product
```bash
curl -X POST "http://localhost:8080/api/products" \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "SKU-013",
    "name": "Wireless Mechanical Numpad",
    "price": 320000
  }'
```

### 3. Update Product
```bash
curl -X PUT "http://localhost:8080/api/products/1" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop ASUS VivoBook 15 Pro",
    "price": 8900000,
    "is_active": true
  }'
```

### 4. Delete Product (Rejected with 409 if has movement history)
```bash
curl -i -X DELETE "http://localhost:8080/api/products/1"
```

### 5. Adjust Stock (In / Out)
```bash
# Stock In
curl -X POST "http://localhost:8080/api/products/1/stock" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "in",
    "qty": 20,
    "note": "Restock batch 2026-Q4"
  }'

# Stock Out (Overdraw returns 409 with available units)
curl -i -X POST "http://localhost:8080/api/products/1/stock" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "out",
    "qty": 9999,
    "note": "Testing overdraw rejection"
  }'
```

### 6. List Stock Movements
```bash
curl -X GET "http://localhost:8080/api/products/1/movements"
```

### 7. Create Sales Order (Atomic multi-item decrement)
```bash
# Successful order
curl -X POST "http://localhost:8080/api/orders" \
  -H "Content-Type: application/json" \
  -d '{
    "customer": "PT Sumber Rejeki",
    "items": [
      { "sku": "SKU-002", "qty": 2 },
      { "sku": "SKU-003", "qty": 1 }
    ]
  }'

# Insufficient stock order (returns 409 with exact SKU and available qty)
curl -i -X POST "http://localhost:8080/api/orders" \
  -H "Content-Type: application/json" \
  -d '{
    "customer": "Overdraw Tester",
    "items": [
      { "sku": "SKU-010", "qty": 999 }
    ]
  }'
```

### 8. Get Order Detail
```bash
curl -X GET "http://localhost:8080/api/orders/1"
```

### 9. Cancel Order (Restores inventory, idempotent 409 on second cancel)
```bash
curl -X POST "http://localhost:8080/api/orders/1/cancel"
```

---

## What I Would Improve Next

Given more time, here are the targeted architectural enhancements:
1. **Idempotency-Key Support on `POST /api/orders`**:
   - Store header `Idempotency-Key: <uuid>` in an `idempotency_keys` table with hashed payload and cached response to guarantee duplicate network retries never double-charge inventory.
2. **Daily Sales & Inventory Reporting**:
   - Add `GET /api/reports/daily-sales?date=YYYY-MM-DD` aggregating revenue, gross order count, and top-selling SKUs for the warehouse supervisor dashboard.
3. **Soft-Delete Archive (`deleted_at`)**:
   - While products with movement history cannot be permanently purged to preserve financial and audit records, a soft-delete status would allow archiving legacy SKUs without breaking historical order item foreign keys.
4. **Read Replica & Query Cache**:
   - For warehouse scale with millions of movement records, add Redis caching or a materialized view refreshed periodically for `last_movement_at` and catalog search.
5. **Cursor-based Pagination**:
   - Replace offset pagination with keyset (`created_at, id`) pagination to avoid degradation on very large product tables.

---

## Honest Hours Spent

Total focused time: **~5.5 hours**

| Phase | Description | Hours |
|---|---|---|
| **Phase 1: Backend Foundation** | Database schema, foreign keys, indexes, seeds, Chi router & repositories | ~1.5 h |
| **Phase 2: Concurrency & Safety** | Atomic SQL update predicates, transaction isolation, Go test suite & race checks | ~1.0 h |
| **Phase 3: Docker & DevOps** | Multi-stage Dockerfiles (Go Alpine, Vite Nginx), docker-compose, auto-migrations | ~0.5 h |
| **Phase 4: Frontend & Vitest** | React 19 + TypeScript, Dark modern UI, search debounce, modal flows, Vitest suite | ~1.5 h |
| **Phase 5: Documentation & Polish**| Cross-stack error verification, cURL test examples, Git commits & README | ~1.0 h |
| **Total** | | **~5.5 h** |
