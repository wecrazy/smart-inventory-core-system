import type { ReportFilters } from '../../app/apiTypes';

type ReportFilterInput = Omit<ReportFilters, 'type'> & {
  type?: '' | 'STOCK_IN' | 'STOCK_OUT';
};

export function sanitizeReportFilters(filters: ReportFilterInput): ReportFilters {
  const sanitized: ReportFilters = {};

  if (filters.type) {
    sanitized.type = filters.type;
  }

  if (filters.referenceCode && filters.referenceCode.trim() !== '') {
    sanitized.referenceCode = filters.referenceCode.trim();
  }

  if (filters.completedFrom) {
    sanitized.completedFrom = filters.completedFrom;
  }

  if (filters.completedTo) {
    sanitized.completedTo = filters.completedTo;
  }

  if (typeof filters.limit === 'number') {
    sanitized.limit = filters.limit;
  }

  if (typeof filters.offset === 'number') {
    sanitized.offset = filters.offset;
  }

  return sanitized;
}

export function createReportExportFilename(now: Date = new Date()): string {
  const year = now.getUTCFullYear();
  const month = String(now.getUTCMonth() + 1).padStart(2, '0');
  const day = String(now.getUTCDate()).padStart(2, '0');
  const hours = String(now.getUTCHours()).padStart(2, '0');
  const minutes = String(now.getUTCMinutes()).padStart(2, '0');
  const seconds = String(now.getUTCSeconds()).padStart(2, '0');

  return `smart-inventory-report-${year}${month}${day}-${hours}${minutes}${seconds}.csv`;
}

export function buildReportExportUrl(baseUrl: string, filters: ReportFilters): string {
  const url = new URL(`${baseUrl.replace(/\/$/, '')}/reports/export`);
  const sanitized = sanitizeReportFilters(filters);

  Object.entries(sanitized).forEach(([key, value]) => {
    if (value !== undefined) {
      url.searchParams.set(key, String(value));
    }
  });

  return url.toString();
}

export function extractExportFilename(contentDisposition: string | null, fallback: string): string {
  if (!contentDisposition) {
    return fallback;
  }

  const match = contentDisposition.match(/filename="?([^";]+)"?/i);

  return match?.[1] ?? fallback;
}