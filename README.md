# Stockflow 📦

A full-stack warehouse management application built for warehouse supervisors to manage products, adjust stock levels, and record sales orders that safely consume inventory under concurrent load.

Built as an end-to-end vertical slice: **PostgreSQL → Go REST API → React + TypeScript (Vite)**.

---

## 🌟 Key Features & Architectural Highlights

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

## 🏗️ Repository Layout

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

## 🚀 How to Run

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
Configured via `backend/.env`:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=1234
DB_NAME=stockflow_db
```

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

## 🧪 Running Tests

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

## 🧠 Design Decisions & Trade-offs

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
