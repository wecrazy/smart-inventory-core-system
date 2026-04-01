import { describe, expect, it } from 'vitest';

import { buildReportExportUrl, createReportExportFilename, extractExportFilename, sanitizeReportFilters } from './reportExport';

describe('reportExport', () => {
  it('sanitizes report filters before they are sent to the API or export endpoint', () => {
    expect(
      sanitizeReportFilters({
        type: 'STOCK_OUT',
        referenceCode: '  OUT-MANUAL  ',
        completedFrom: '2026-04-01',
        completedTo: '',
        limit: 10,
        offset: 0,
      }),
    ).toEqual({
      type: 'STOCK_OUT',
      referenceCode: 'OUT-MANUAL',
      completedFrom: '2026-04-01',
      limit: 10,
      offset: 0,
    });
  });

  it('creates a stable export filename from a provided timestamp', () => {
    expect(createReportExportFilename(new Date('2026-04-01T13:14:15Z'))).toBe('smart-inventory-report-20260401-131415.csv');
  });

  it('builds a filtered report export URL and extracts the server filename when present', () => {
    expect(
      buildReportExportUrl('http://localhost:8080/api/v1', {
        type: 'STOCK_OUT',
        referenceCode: 'OUT-MANUAL',
        completedFrom: '2026-04-01',
        completedTo: '2026-04-02',
      }),
    ).toBe('http://localhost:8080/api/v1/reports/export?type=STOCK_OUT&referenceCode=OUT-MANUAL&completedFrom=2026-04-01&completedTo=2026-04-02');

    expect(extractExportFilename('attachment; filename="filtered-report.csv"', 'fallback.csv')).toBe('filtered-report.csv');
    expect(extractExportFilename(null, 'fallback.csv')).toBe('fallback.csv');
  });
});