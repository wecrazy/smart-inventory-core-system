# Smart Inventory Core System

Implementasi technical assessment untuk sistem manajemen stok dengan audit trail transaksi, reservasi stok keluar yang aman, dan antarmuka operator berbasis React.

Versi English disediakan sebagai pelengkap di `README.en.md`, tetapi file `README.md` ini disusun dalam Bahasa Indonesia agar sesuai dengan requirement assessment.

## Ringkasan

### Stack

| Layer | Teknologi |
| --- | --- |
| Backend | Go 1.26, Fiber v3, pgx, PostgreSQL |
| Frontend | React 18, Vite, Redux Toolkit, RTK Query, React Router |
| Tooling | Make, Vitest, Testing Library |

### Fitur Yang Sudah Diimplementasikan

- Menampilkan inventory dengan filter berdasarkan nama, SKU, dan customer.
- Memisahkan `physical stock`, `reserved stock`, dan `available stock`.
- Menyimpan stock adjustment sebagai transaksi yang dapat diaudit.
- Mendukung alur stock-in: `CREATED -> IN_PROGRESS -> DONE` dengan opsi cancel sebelum selesai.
- Mendukung alur stock-out: `ALLOCATED -> IN_PROGRESS -> DONE` dengan rollback reservasi saat dibatalkan.
- Menyediakan report yang hanya menampilkan transaksi stock-in dan stock-out dengan status `DONE`.

## Struktur Repository

```text
backend/
  cmd/api/
  docs/swagger/
  internal/app/
  internal/domain/
  internal/platform/
frontend/
```

## Kebutuhan Sistem

- Go 1.26+
- PostgreSQL 15+
- Node.js 20+ dan npm
- make
- psql

## Arsitektur Singkat

Backend menggunakan pendekatan berlapis:

- `internal/domain` berisi model bisnis dan aturan transisi status.
- `internal/app/service` berisi validasi input, normalisasi data, dan orkestrasi use case.
- `internal/platform/postgres` berisi implementasi repository PostgreSQL, transaksi database, dan row locking.
- `internal/app/http` berisi router, handler HTTP, dan dokumentasi Swagger.

Frontend menggunakan Redux Toolkit + RTK Query:

- `frontend/src/app/store.ts` menjadi pusat state management.
- `frontend/src/app/baseApi.ts` dan file endpoint terpisah menangani komunikasi API.
- Feature page dipisah berdasarkan domain: Inventory, Stock In, Stock Out, dan Reports.

Diagram Mermaid di bawah ini ditambahkan untuk membantu pembaca GitHub memahami boundary sistem dan alur status inti tanpa harus menafsirkan semua penjelasan tekstual terlebih dahulu.

### Diagram Arsitektur

```mermaid
flowchart LR
  subgraph FE[Frontend React]
    Pages[Pages\nInventory | Stock In | Stock Out | Reports]
    State[Redux Store + RTK Query]
    Pages --> State
  end

  subgraph BE[Backend Go Fiber]
    HTTP[HTTP Router + Handlers]
    Service[Service Layer\nvalidasi + orkestrasi use case]
    Domain[Domain Rules\ntransisi status + audit rule]
    Repo[PostgreSQL Repository\ntransaction + row locking]
    HTTP --> Service
    Service --> Domain
    Service --> Repo
  end

  DB[(PostgreSQL)]
  Swagger[Swagger UI]

  State --> HTTP
  Repo --> DB
  HTTP --> Swagger
```

### Diagram Workflow Status

#### Stock-In

```mermaid
stateDiagram-v2
  [*] --> CREATED
  CREATED --> IN_PROGRESS: operator mulai proses
  CREATED --> CANCELLED: dibatalkan sebelum selesai
  IN_PROGRESS --> DONE: stok fisik ditambahkan
  IN_PROGRESS --> CANCELLED: dibatalkan sebelum selesai
  DONE --> [*]
  CANCELLED --> [*]

  note right of DONE
    physical_stock bertambah
    masuk ke report DONE
  end note
```

#### Stock-Out

```mermaid
stateDiagram-v2
  [*] --> ALLOCATED: reserve available stock
  ALLOCATED --> IN_PROGRESS: operator mulai picking
  ALLOCATED --> CANCELLED: rollback reservasi
  IN_PROGRESS --> DONE: stok keluar difinalkan
  IN_PROGRESS --> CANCELLED: rollback reservasi
  DONE --> [*]
  CANCELLED --> [*]

  note right of ALLOCATED
    reserved_stock naik
    available_stock turun
  end note

  note right of DONE
    physical_stock turun
    masuk ke report DONE
  end note
```

`ADJUSTMENT` tetap disimpan sebagai transaksi audit, tetapi tidak masuk report karena report hanya mengambil stock-in dan stock-out dengan status `DONE`.

## Keputusan Arsitektur

### Konteks

Assessment ini membutuhkan transisi status yang ketat, report yang hanya menampilkan transaksi selesai, dan sinkronisasi state yang konsisten antara backend dan frontend. Risiko teknis tertinggi ada pada proses stock-out, karena reservasi stok harus aman saat ada request paralel.

### Keputusan Desain

- API backend menggunakan Go Fiber v3.
- PostgreSQL diakses melalui `pgx`.
- Repository PostgreSQL menangani mutasi stok dengan transaction database dan row locking.
- Frontend menggunakan React dengan Redux Toolkit dan RTK Query.
- Repository disusun sebagai monorepo dengan `backend/` dan `frontend/`.

Model inventory dipisahkan menjadi `physical_stock`, `reserved_stock`, dan `available_stock = physical_stock - reserved_stock`.

Jenis transaksi yang digunakan adalah `STOCK_IN`, `STOCK_OUT`, dan `ADJUSTMENT`, sedangkan riwayat status disimpan terpisah agar audit trail tetap eksplisit.

### Konsekuensi

Keuntungan utama:

- alur reservasi dan rollback stock-out tetap aman dan dapat diaudit
- report menjadi sederhana karena hanya transaksi `DONE` yang ditampilkan
- frontend dapat langsung mengonsumsi API yang fokus pada workflow operator

Tradeoff utama:

- bootstrap database masih berbasis schema SQL, belum migration versioned
- `customer_name` masih disimpan langsung di inventory item
- belum ada test concurrency pada level repository/database

## Ringkasan Frontend

### Tujuan UI

Frontend berfungsi sebagai operator UI untuk melihat inventory, membuat inventory baru, melakukan stock adjustment, menjalankan workflow stock-in, menjalankan workflow stock-out, dan melihat report transaksi selesai.

### Rute Frontend

- `/` untuk inventory search, create inventory, dan stock adjustment
- `/stock-in` untuk pembuatan transaksi masuk dan perubahan statusnya
- `/stock-out` untuk alokasi, proses, cancel, dan penyelesaian transaksi keluar
- `/reports` untuk report transaksi selesai

### Data Flow Frontend

- RTK Query digunakan sebagai lapisan HTTP tunggal
- endpoint frontend dipisah per domain di `frontend/src/app/`
- UI mengasumsikan backend mengembalikan envelope `{ data, error? }`

### Dependensi Backend Untuk Frontend

Frontend menggunakan endpoint berikut:

- `GET /api/v1/inventory`
- `POST /api/v1/inventory`
- `POST /api/v1/inventory/adjustments`
- `POST /api/v1/stock-in`
- `GET /api/v1/stock-in`
- `PATCH /api/v1/stock-in/:id/status`
- `POST /api/v1/stock-in/:id/cancel`
- `POST /api/v1/stock-out`
- `GET /api/v1/stock-out`
- `PATCH /api/v1/stock-out/:id/status`
- `POST /api/v1/stock-out/:id/cancel`
- `GET /api/v1/reports`
- `GET /api/v1/reports/export`

### Status Testing Frontend

- test frontend saat ini mencakup app shell dan utilitas API sederhana
- coverage alur form dan mutation feedback masih bisa diperdalam

## Quick Start

Jalankan dari root repository:

```bash
git clone https://github.com/wecrazy/smart-inventory-core-system.git
cd smart-inventory-core-system
make env
make install
make schema
make dev
```

Clone repository dari:

- `https://github.com/wecrazy/smart-inventory-core-system.git`

URL aplikasi:

- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080/api/v1`
- Swagger UI: `http://localhost:8080/swagger/index.html`

Setelah `make dev`, ikuti panduan uji manual di [TESTING_GUIDE.md](TESTING_GUIDE.md) untuk langkah operator dari setup inventory, stock adjustment, stock-in, stock-out, sampai report. Versi English tersedia di [TESTING_GUIDE.en.md](TESTING_GUIDE.en.md).

## Make Targets

| Command | Fungsi |
| --- | --- |
| `make env` | Membuat file env lokal jika belum ada |
| `make install` | Menginstal dependency backend dan frontend |
| `make db-create` | Membuat database PostgreSQL dari `backend/.env` jika belum ada |
| `make db-drop` | Menghapus database PostgreSQL dari `backend/.env` jika ada |
| `make db-reset` | Menghapus, membuat ulang, dan meng-apply schema database |
| `make schema` | Membuat database bila perlu lalu meng-apply schema PostgreSQL |
| `make backend-run` | Menjalankan API Go |
| `make frontend-run` | Menjalankan frontend Vite |
| `make dev` | Menjalankan backend dan frontend bersamaan |
| `make backend-fmt` | Mengecek apakah file Go backend sudah terformat |
| `make backend-vet` | Menjalankan `go vet` untuk backend |
| `make backend-revive` | Menjalankan `revive` menggunakan `backend/revive.toml` |
| `make backend-lint` | Menjalankan format check, `go vet`, dan `revive` untuk backend |
| `make lint` | Alias untuk lint backend |
| `make test` | Menjalankan test backend dan frontend |
| `make backend-docs` | Regenerate dokumen Swagger dari anotasi Go |
| `make build` | Build binary backend dan bundle frontend |
| `make clean` | Menghapus artefak build dan test |

## Environment Files

File env lokal yang digunakan:

- `backend/.env`
- `frontend/.env`

Default backend:

```env
APP_ENV=development
HTTP_PORT=8080
DATABASE_URL=postgres://wegil:postgres@localhost:5432/smart_inventory?sslmode=disable
```

Default frontend:

```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

## Setup Manual

### Backend

```bash
set -a
source backend/.env
set +a

make schema
cd backend
go mod tidy
go run ./cmd/api
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Verifikasi

Sudah diverifikasi dengan:

- `cd backend && go test ./...`
- `make backend-docs`
- `cd frontend && npm run test`
- `cd frontend && npm run build`
- `make test`
- `make build`

## Dokumentasi API

- Dokumentasi API backend digenerate dari anotasi handler dan disajikan melalui Swagger UI.

## API Surface

### Inventory

- `GET /api/v1/inventory`
- `POST /api/v1/inventory`
- `POST /api/v1/inventory/adjustments`

### Stock In

- `POST /api/v1/stock-in`
- `GET /api/v1/stock-in`
- `GET /api/v1/stock-in/:id`
- `PATCH /api/v1/stock-in/:id/status`
- `POST /api/v1/stock-in/:id/cancel`

### Stock Out

- `POST /api/v1/stock-out`
- `GET /api/v1/stock-out`
- `GET /api/v1/stock-out/:id`
- `PATCH /api/v1/stock-out/:id/status`
- `POST /api/v1/stock-out/:id/cancel`

### Reports

- `GET /api/v1/reports`
- `GET /api/v1/reports/export`

## Tradeoff Saat Ini

- Bootstrap database masih menggunakan file schema SQL, belum migrasi versi seperti Goose atau Atlas.
- Test frontend masih level smoke test, belum mencakup alur form dan mutasi secara mendalam.
- Mekanisme concurrency backend sudah memakai row locking, tetapi belum ada test kompetisi transaksi pada level repository.
