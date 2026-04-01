import { FormEvent, useState } from 'react';

import type { ReportFilters } from '../../app/apiTypes';
import { useGetReportsQuery } from '../../app/api';
import { apiBaseUrl } from '../../app/baseApi';
import { FeedbackBanner } from '../../components/FeedbackBanner';
import type { ReportPage, Transaction } from '../../types';

import { buildReportExportUrl, createReportExportFilename, extractExportFilename, sanitizeReportFilters } from './reportExport';

type FeedbackState = {
  tone: 'success' | 'error';
  message: string;
};

type ReportFilterFormState = {
  type: '' | 'STOCK_IN' | 'STOCK_OUT';
  referenceCode: string;
  completedFrom: string;
  completedTo: string;
};

const reportPageSize = 10;
const initialReportFilters: ReportFilterFormState = {
  type: '',
  referenceCode: '',
  completedFrom: '',
  completedTo: '',
};

const emptyReportPage: ReportPage = {
  items: [],
  total: 0,
  limit: reportPageSize,
  offset: 0,
  unitsIn: 0,
  unitsOut: 0,
};

export function ReportsPage() {
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const [offset, setOffset] = useState(0);
  const [isExporting, setIsExporting] = useState(false);
  const [draftFilters, setDraftFilters] = useState<ReportFilterFormState>(initialReportFilters);
  const [activeFilters, setActiveFilters] = useState<ReportFilters>({});
  const [expandedReports, setExpandedReports] = useState<Record<number, boolean>>({});
  const reportQuery = {
    ...activeFilters,
    limit: reportPageSize,
    offset,
  };
  const { data: reportPage = emptyReportPage, isLoading, isFetching, error } = useGetReportsQuery({
    ...reportQuery,
  });
  const reports = reportPage.items;
  const totalLineItems = reports.reduce((sum, report) => sum + report.items.length, 0);
  const hasReports = reports.length > 0;
  const hasMatchingReports = reportPage.total > 0;
  const hasActiveFilters = Object.keys(activeFilters).length > 0;
  const firstVisibleItem = reportPage.total === 0 ? 0 : reportPage.offset + 1;
  const lastVisibleItem = reportPage.offset + reports.length;
  const totalPages = reportPage.total === 0 ? 1 : Math.ceil(reportPage.total / reportPage.limit);
  const currentPage = reportPage.total === 0 ? 1 : Math.floor(reportPage.offset / reportPage.limit) + 1;
  const hasPreviousPage = reportPage.offset > 0;
  const hasNextPage = reportPage.offset + reports.length < reportPage.total;

  function handleApplyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setActiveFilters(sanitizeReportFilters(draftFilters));
    setOffset(0);
    setExpandedReports({});
  }

  function handleClearFilters() {
    setDraftFilters(initialReportFilters);
    setActiveFilters({});
    setOffset(0);
    setExpandedReports({});
  }

  function handlePrint() {
    if (!hasReports || typeof window === 'undefined') {
      return;
    }

    window.print();
  }

  async function handleExportCsv() {
    if (!hasMatchingReports || typeof document === 'undefined' || typeof URL === 'undefined' || typeof URL.createObjectURL !== 'function') {
      return;
    }

    try {
      setIsExporting(true);
      const response = await fetch(buildReportExportUrl(apiBaseUrl, activeFilters), {
        headers: {
          Accept: 'text/csv',
        },
      });

      if (!response.ok) {
        let message = 'Failed to export the detailed report.';

        try {
          const errorPayload = (await response.json()) as { error?: string };
          if (typeof errorPayload.error === 'string' && errorPayload.error.trim() !== '') {
            message = errorPayload.error;
          }
        } catch {
          // Ignore JSON parsing errors and keep the fallback message.
        }

        throw new Error(message);
      }

      const blob = await response.blob();
      const objectUrl = URL.createObjectURL(blob);
      const link = document.createElement('a');

      link.href = objectUrl;
      link.download = extractExportFilename(response.headers.get('Content-Disposition'), createReportExportFilename());
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(objectUrl);

      setFeedback({ tone: 'success', message: 'Filtered report export completed.' });
    } catch (exportError) {
      setFeedback({
        tone: 'error',
        message: exportError instanceof Error ? exportError.message : 'Failed to export the detailed report.',
      });
    } finally {
      setIsExporting(false);
    }
  }

  function handleToggleReport(reportID: number) {
    setExpandedReports((current) => ({
      ...current,
      [reportID]: !current[reportID],
    }));
  }

  function handlePageChange(nextOffset: number) {
    setOffset(nextOffset);
    setExpandedReports({});
  }

  return (
    <section className="panel panel-wide stack report-print-root">
      <div className="panel-header report-header">
        <div>
          <p className="eyebrow">Done-only reporting</p>
          <h2>Completed transaction report</h2>
        </div>
        <div className="button-row report-actions">
          <button className="button button-ghost" disabled={!hasReports || isFetching} onClick={handlePrint} type="button">
            Print report
          </button>
          <button className="button" disabled={!hasMatchingReports || isFetching || isExporting} onClick={() => void handleExportCsv()} type="button">
            {isExporting ? 'Exporting…' : 'Export CSV'}
          </button>
        </div>
      </div>
      <p className="helper-copy">Halaman ini hanya menampilkan stock-in dan stock-out dengan status DONE. Adjustment tidak masuk ke report, dan transaksi yang belum selesai memang tidak akan muncul di sini.</p>
      <p className="helper-copy">Report dimuat 10 transaksi per halaman dari server agar halaman tetap ringan. Print menggunakan data halaman yang sedang ditampilkan, sedangkan Export CSV mengambil semua report yang cocok dengan filter aktif dari server.</p>
      <form className="panel report-filter-form stack" onSubmit={handleApplyFilters}>
        <div>
          <p className="eyebrow">Report filters</p>
          <h3 className="section-title">Filter transaksi selesai</h3>
        </div>
        <div className="report-filter-grid">
          <label className="stack field-group">
            <span className="meta-label">Type</span>
            <select className="field" value={draftFilters.type} onChange={(event) => setDraftFilters((current) => ({ ...current, type: event.target.value as ReportFilterFormState['type'] }))}>
              <option value="">All types</option>
              <option value="STOCK_IN">Stock In</option>
              <option value="STOCK_OUT">Stock Out</option>
            </select>
          </label>
          <label className="stack field-group">
            <span className="meta-label">Reference code</span>
            <input className="field" placeholder="OUT-MANUAL" value={draftFilters.referenceCode} onChange={(event) => setDraftFilters((current) => ({ ...current, referenceCode: event.target.value }))} />
          </label>
          <label className="stack field-group">
            <span className="meta-label">Completed from</span>
            <input className="field" type="date" value={draftFilters.completedFrom} onChange={(event) => setDraftFilters((current) => ({ ...current, completedFrom: event.target.value }))} />
          </label>
          <label className="stack field-group">
            <span className="meta-label">Completed to</span>
            <input className="field" type="date" value={draftFilters.completedTo} onChange={(event) => setDraftFilters((current) => ({ ...current, completedTo: event.target.value }))} />
          </label>
        </div>
        <div className="button-row report-filter-actions">
          <button className="button" disabled={isFetching} type="submit">
            Apply filters
          </button>
          <button className="button button-ghost" disabled={isFetching && !hasActiveFilters} onClick={handleClearFilters} type="button">
            Clear filters
          </button>
        </div>
      </form>
      {feedback ? <FeedbackBanner message={feedback.message} onClose={() => setFeedback(null)} tone={feedback.tone} /> : null}
      {isLoading ? <p>Loading reports…</p> : null}
      {error ? <p className="error-copy">Failed to load reports.</p> : null}
      {!isLoading && !error && reports.length > 0 ? (
        <div className="report-summary-grid">
          <div className="report-summary-card">
            <span className="meta-label">Matching transactions</span>
            <strong className="meta-value">{reportPage.total}</strong>
          </div>
          <div className="report-summary-card">
            <span className="meta-label">Units moved in</span>
            <strong className="meta-value">{reportPage.unitsIn}</strong>
          </div>
          <div className="report-summary-card">
            <span className="meta-label">Units moved out</span>
            <strong className="meta-value">{reportPage.unitsOut}</strong>
          </div>
          <div className="report-summary-card">
            <span className="meta-label">Line items on page</span>
            <strong className="meta-value">{totalLineItems}</strong>
          </div>
        </div>
      ) : null}
      {!isLoading && !error && reportPage.total > 0 ? (
        <div className="report-pagination">
          <p className="muted-copy">
            Showing {firstVisibleItem}-{lastVisibleItem} of {reportPage.total} transactions. Page {currentPage} of {totalPages}.
          </p>
          <div className="button-row report-pagination-actions">
            <button className="button button-ghost" disabled={!hasPreviousPage || isFetching} onClick={() => handlePageChange(Math.max(reportPage.offset - reportPage.limit, 0))} type="button">
              Previous page
            </button>
            <button className="button button-ghost" disabled={!hasNextPage || isFetching} onClick={() => handlePageChange(reportPage.offset + reportPage.limit)} type="button">
              Next page
            </button>
          </div>
        </div>
      ) : null}
      <div className="stack">
        {!isLoading && !error && reports.length === 0 ? (
          <div className="empty-state">
            {hasActiveFilters
              ? 'Tidak ada report DONE yang cocok dengan filter saat ini. Ubah atau kosongkan filter untuk melihat transaksi selesai lainnya.'
              : 'Belum ada report yang dapat ditampilkan. Selesaikan minimal satu transaksi stock-in atau stock-out sampai status DONE agar report muncul di sini.'}
          </div>
        ) : null}
        {reports.map((report) => (
          <ReportCard
            expanded={Boolean(expandedReports[report.id])}
            key={report.id}
            onToggle={() => handleToggleReport(report.id)}
            report={report}
          />
        ))}
      </div>
    </section>
  );
}

function ReportCard({
  expanded,
  onToggle,
  report,
}: {
  expanded: boolean;
  onToggle: () => void;
  report: Transaction;
}) {
  const totalQuantity = report.items.reduce((sum, item) => sum + item.quantity, 0);

  return (
    <article className={`transaction-card report-card ${expanded ? 'report-card-expanded' : 'report-card-collapsed'}`}>
      <div className="transaction-head report-card-head">
        <div>
          <h3>{report.referenceCode}</h3>
          <p className="muted-copy">Laporan detail transaksi {formatTransactionType(report.type)} yang sudah selesai diproses.</p>
        </div>
        <div className="report-card-controls">
          <span className={`status-pill status-${report.status.toLowerCase()}`}>{report.status}</span>
          <button aria-expanded={expanded} className="button button-ghost report-toggle" onClick={onToggle} type="button">
            {expanded ? 'Hide details' : 'Show details'}
          </button>
        </div>
      </div>

      <dl className="report-compact-grid">
        <div className="report-meta-block">
          <dt className="meta-label">Transaction ID</dt>
          <dd className="meta-value">#{report.id}</dd>
        </div>
        <div className="report-meta-block">
          <dt className="meta-label">Type</dt>
          <dd className="meta-value">{formatTransactionType(report.type)}</dd>
        </div>
        <div className="report-meta-block">
          <dt className="meta-label">Completed at</dt>
          <dd className="meta-value">{formatDateTime(report.completedAt)}</dd>
        </div>
        <div className="report-meta-block">
          <dt className="meta-label">Total quantity</dt>
          <dd className="meta-value">{totalQuantity}</dd>
        </div>
        <div className="report-meta-block report-meta-wide">
          <dt className="meta-label">Operator note</dt>
          <dd className="meta-value">{report.note || 'No note recorded.'}</dd>
        </div>
      </dl>

      <div className="report-body stack">
        <dl className="report-meta-grid">
          <div className="report-meta-block">
            <dt className="meta-label">Created at</dt>
            <dd className="meta-value">{formatDateTime(report.createdAt)}</dd>
          </div>
          <div className="report-meta-block">
            <dt className="meta-label">Completed at</dt>
            <dd className="meta-value">{formatDateTime(report.completedAt)}</dd>
          </div>
          <div className="report-meta-block">
            <dt className="meta-label">Line count</dt>
            <dd className="meta-value">{report.items.length}</dd>
          </div>
        </dl>

        <section className="report-section stack">
          <div>
            <p className="eyebrow">Line items</p>
            <h4 className="section-title">Detail item per transaksi</h4>
          </div>
          <div className="table-wrap">
            <table className="detail-table">
              <thead>
                <tr>
                  <th>Inventory ID</th>
                  <th>SKU</th>
                  <th>Name</th>
                  <th>Customer</th>
                  <th>Quantity</th>
                </tr>
              </thead>
              <tbody>
                {report.items.map((item) => (
                  <tr key={`${report.id}-${item.inventoryId}`}>
                    <td>{item.inventoryId}</td>
                    <td>{item.sku}</td>
                    <td>{item.name}</td>
                    <td>{item.customerName}</td>
                    <td>{item.quantity}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section className="report-section stack">
          <div>
            <p className="eyebrow">Status history</p>
            <h4 className="section-title">Riwayat proses transaksi</h4>
          </div>
          <ul className="history-list">
            {report.history.map((entry, index) => (
              <li className="history-item" key={`${report.id}-${entry.status}-${entry.createdAt}-${index}`}>
                <div className="history-head">
                  <span className={`status-pill status-${entry.status.toLowerCase()}`}>{entry.status}</span>
                  <span className="muted-copy">{formatDateTime(entry.createdAt)}</span>
                </div>
                <p className="muted-copy">{entry.note || 'No note recorded for this status change.'}</p>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </article>
  );
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return '-';
  }

  return new Date(value).toLocaleString();
}

function formatTransactionType(type: Transaction['type']) {
  switch (type) {
    case 'STOCK_IN':
      return 'Stock In';
    case 'STOCK_OUT':
      return 'Stock Out';
    case 'ADJUSTMENT':
      return 'Adjustment';
    default:
      return type;
  }
}