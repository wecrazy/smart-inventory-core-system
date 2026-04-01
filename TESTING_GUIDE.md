# Panduan Uji Manual

Panduan ini menjelaskan langkah operator untuk menguji aplikasi dari frontend sampai report akhir sesuai instruksi assessment.

## 1. Jalankan Aplikasi

Jalankan dari root repository:

```bash
make env
make install
make schema
make dev
```

Setelah itu buka:

- Frontend: `http://localhost:5173`
- Swagger UI: `http://localhost:8080/swagger/index.html`

## 2. Siapkan Data Inventory

Masuk ke halaman `Inventory`.

Tambahkan minimal dua item berikut agar alur stock-in, stock-out, dan report bisa diuji:

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

Hasil yang harus terlihat:

- tabel inventory menampilkan item yang baru dibuat
- `physical stock` sesuai nilai awal
- `reserved stock` masih `0`
- `available stock` sama dengan `physical stock`

Jika tabel kosong karena filter aktif, kosongkan kolom search, SKU, dan customer.

## 3. Uji Stock Adjustment

Masih di halaman `Inventory`, gunakan panel `Stock adjustment`.

Contoh uji:

- pilih `SKU-001`
- isi `New physical stock` menjadi `120`
- isi `Reference code` dengan `ADJ-MANUAL-001`
- isi note bebas, misalnya `Cycle count correction`

Hasil yang harus terlihat:

- notifikasi sukses muncul di frontend
- `physical stock` item berubah menjadi `120`
- `available stock` ikut berubah menjadi `120`
- perubahan ini tercatat sebagai transaksi adjustment yang dapat diaudit di backend, tetapi tidak tampil di halaman report karena report hanya untuk stock-in dan stock-out yang `DONE`

## 4. Uji Alur Stock In

Masuk ke halaman `Stock In`.

Buat transaksi baru:

- reference code: `IN-MANUAL-001`
- note: `Penerimaan pagi`
- item: `SKU-001`
- quantity: `10`

Setelah tersimpan, transaksi akan muncul dengan status `CREATED`.

Lanjutkan uji status:

1. Klik `Move to in progress`
2. Pastikan status berubah menjadi `IN_PROGRESS`
3. Cek kembali halaman `Inventory`
4. Pastikan `physical stock` belum bertambah saat masih `IN_PROGRESS`
5. Kembali ke halaman `Stock In`
6. Klik `Mark done`

Hasil yang harus terlihat:

- status berubah menjadi `DONE`
- `physical stock` item bertambah `10`
- `available stock` ikut bertambah `10`
- transaksi tersebut mulai muncul di halaman `Reports`

## 5. Uji Alur Stock Out Dua Tahap

Masuk ke halaman `Stock Out`.

Buat transaksi baru:

- reference code: `OUT-MANUAL-001`
- note: `Pengiriman batch 1`
- item: `SKU-001`
- quantity: `15`

Hasil tahap allocation:

- transaksi dibuat dengan status `ALLOCATED`
- di halaman `Inventory`, `reserved stock` bertambah `15`
- `available stock` berkurang `15`
- `physical stock` belum berubah

Lanjutkan uji dua skenario berikut.

### Skenario A: Rollback Saat Cancel

1. Pada transaksi `OUT-MANUAL-001`, klik `Move to in progress`
2. Pastikan status berubah menjadi `IN_PROGRESS`
3. Klik `Cancel and rollback`

Hasil yang harus terlihat:

- status menjadi `CANCELLED`
- `reserved stock` kembali turun
- `available stock` kembali seperti sebelum alokasi
- `physical stock` tetap tidak berubah
- transaksi ini tidak muncul di halaman `Reports`

### Skenario B: Selesaikan Sampai DONE

Buat transaksi baru lagi:

- reference code: `OUT-MANUAL-002`
- note: `Pengiriman final`
- item: `SKU-001`
- quantity: `20`

Lalu:

1. Klik `Move to in progress`
2. Klik `Mark done`

Hasil yang harus terlihat:

- status menjadi `DONE`
- `physical stock` berkurang `20`
- `reserved stock` kembali turun sesuai kuantitas yang selesai
- transaksi ini muncul di halaman `Reports`

## 6. Uji Reports

Masuk ke halaman `Reports`.

Halaman ini hanya menampilkan transaksi:

- `STOCK_IN` dengan status `DONE`
- `STOCK_OUT` dengan status `DONE`

Yang tidak boleh tampil:

- adjustment
- stock-in yang masih `CREATED`, `IN_PROGRESS`, atau `CANCELLED`
- stock-out yang masih `ALLOCATED`, `IN_PROGRESS`, atau `CANCELLED`

Jika belum ada data report, frontend sekarang menampilkan empty state yang menjelaskan bahwa transaksi harus diselesaikan ke `DONE` terlebih dahulu.

Jika data report sudah ada, perhatikan juga perilaku berikut:

- halaman report memuat `10` transaksi selesai per halaman dari server, bukan mengambil seluruh report sekaligus
- operator bisa memfilter report berdasarkan tipe transaksi, potongan reference code, serta rentang tanggal selesai
- ringkasan report sekarang memisahkan `Units moved in` dan `Units moved out`
- setiap kartu transaksi report tampil dalam keadaan collapse terlebih dahulu, lalu operator bisa klik `Show details` untuk membuka detail item dan riwayat status
- saat klik `Print report`, output print sekarang fokus ke section report, bukan seluruh layout aplikasi

Jika sudah ada data report, operator sekarang juga bisa:

- klik `Print report` untuk membuka dialog print browser dengan layout report yang ramah cetak
- klik `Export CSV` untuk mengunduh semua report yang cocok dengan filter aktif dari server ke file CSV

## 7. Checklist Kesesuaian Dengan Assessment

Checklist verifikasi cepat:

- [ ] inventory bisa dicari berdasarkan nama, SKU, dan customer
- [ ] inventory memisahkan `physical`, `reserved`, dan `available`
- [ ] stock adjustment bisa dilakukan dari UI
- [ ] stock-in mengikuti alur `CREATED -> IN_PROGRESS -> DONE`
- [ ] stock-in bisa di-cancel sebelum `DONE`
- [ ] stock-out mengikuti alur `ALLOCATED -> IN_PROGRESS -> DONE`
- [ ] cancel stock-out me-release reservasi
- [ ] report hanya menampilkan transaksi `DONE`
- [ ] report detail transaksi bisa di-print dan di-export ke CSV
- [ ] report dimuat bertahap dari server dan detail transaksi dibuka lewat accordion
- [ ] report bisa difilter berdasarkan tipe, reference code, dan tanggal selesai
- [ ] frontend menampilkan pesan kosong dan pesan error/sukses yang cukup jelas untuk operator
