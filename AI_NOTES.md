# AI Notes

## Tools AI Yang Digunakan

- GitHub Copilot Chat di VS Code
- Model yang digunakan di sesi implementasi ini: GPT-5.4

## Ruang Penggunaan AI

AI digunakan untuk mempercepat beberapa bagian berikut:

- scaffolding struktur backend dan frontend
- drafting schema PostgreSQL
- pembuatan boilerplate handler, service, dan repository
- drafting dokumentasi README, ADR, dan Swagger annotation
- membantu validasi error build dan test saat integrasi frontend-backend

## Prompt Paling Kompleks Yang Digunakan

Salah satu prompt paling kompleks yang digunakan selama pengerjaan adalah versi ringkas berikut:

> Baca dan analisis mendalam assessment PDF Smart Inventory Core System, lalu implementasikan solusi monorepo dengan backend Go Fiber v3 + PostgreSQL dan frontend React. Pastikan alur stock-in hanya menambah physical stock saat DONE, stock-out menggunakan reservasi dua tahap dengan rollback saat cancel, inventory memisahkan physical/reserved/available stock, report hanya menampilkan transaksi DONE, dan dokumentasikan arsitektur, Makefile, env, test, serta Swagger.

## Bagian Kode AI Yang Dimodifikasi Manual Demi Best Practice

Bagian yang paling signifikan dimodifikasi manual adalah alur transaksi stock-out di repository PostgreSQL.

Perubahan manual yang dilakukan:

- memastikan proses reservasi stok, perubahan status transaksi, dan rollback cancel dijalankan dalam transaction database yang konsisten
- menambahkan row locking (`FOR UPDATE`) agar stok tidak terpakai ganda saat ada request paralel
- memastikan `reserved_stock` hanya dilepas saat cancel, dan `physical_stock` hanya berkurang absolut saat transaksi stock-out menjadi `DONE`

Area kode terkait berada di `backend/internal/platform/postgres/repository.go`.

## Review Manual Yang Diterapkan

- validasi transisi status stock-in dan stock-out
- pemisahan `physical_stock`, `reserved_stock`, dan `available_stock`
- error mapping HTTP agar conflict/not found/validation tetap konsisten
- penggantian integrasi Swagger ke middleware Fiber v3 yang resmi
- penyesuaian Makefile untuk bootstrap database lokal

## Verifikasi Yang Sudah Dilakukan

- `cd backend && go test ./...`
- `cd frontend && npm run test`
- `cd frontend && npm run build`
- `make test`
- `make build`
- `make db-create && make schema`

## Catatan Sisa Tradeoff

- belum ada migration tool versioned; saat ini masih memakai schema SQL langsung
- test concurrency database level masih belum dibuat
- coverage frontend masih fokus pada smoke-level verification, belum end-to-end flow
