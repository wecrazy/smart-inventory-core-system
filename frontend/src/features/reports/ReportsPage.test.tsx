import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ReportsPage } from './ReportsPage';

const mockUseGetReportsQuery = vi.fn();
const mockFetch = vi.fn();

vi.mock('../../app/api', () => ({
  useGetReportsQuery: (...args: unknown[]) => mockUseGetReportsQuery(...args),
}));

describe('ReportsPage', () => {
  beforeEach(() => {
    mockUseGetReportsQuery.mockReset();
    mockFetch.mockReset();
    window.print = vi.fn();
    vi.stubGlobal('fetch', mockFetch);
    URL.createObjectURL = vi.fn(() => 'blob:report-export');
    URL.revokeObjectURL = vi.fn();
    HTMLAnchorElement.prototype.click = vi.fn();
  });

  it('renders detailed report sections per completed transaction', () => {
    mockUseGetReportsQuery.mockReturnValue({
      data: {
        items: [
          {
            id: 17,
            type: 'STOCK_OUT',
            status: 'DONE',
            referenceCode: 'OUT-MANUAL-002',
            note: 'Pengiriman final',
            completedAt: '2026-04-01T12:15:00Z',
            createdAt: '2026-04-01T09:00:00Z',
            updatedAt: '2026-04-01T12:15:00Z',
            items: [
              {
                inventoryId: 1,
                sku: 'SKU-001',
                name: 'Widget A',
                customerName: 'Acme Corp',
                quantity: 20,
              },
            ],
            history: [
              {
                status: 'ALLOCATED',
                note: 'Pengiriman batch 1',
                createdAt: '2026-04-01T10:00:00Z',
              },
              {
                status: 'IN_PROGRESS',
                note: 'Packing started',
                createdAt: '2026-04-01T11:00:00Z',
              },
              {
                status: 'DONE',
                note: 'Delivered to customer',
                createdAt: '2026-04-01T12:15:00Z',
              },
            ],
          },
        ],
        total: 1,
        limit: 10,
        offset: 0,
        unitsIn: 0,
        unitsOut: 20,
      },
      isLoading: false,
      isFetching: false,
      error: undefined,
    });

    render(<ReportsPage />);

    expect(mockUseGetReportsQuery).toHaveBeenCalledWith({ limit: 10, offset: 0 });
    expect(screen.getByText('OUT-MANUAL-002')).toBeInTheDocument();
    expect(screen.getByText('Units moved out')).toBeInTheDocument();
    const toggleButton = screen.getByRole('button', { name: 'Show details' });

    expect(toggleButton).toHaveAttribute('aria-expanded', 'false');

    fireEvent.click(toggleButton);

    expect(toggleButton).toHaveAttribute('aria-expanded', 'true');

    const reportCard = screen.getByText('OUT-MANUAL-002').closest('article');

    expect(reportCard).not.toBeNull();

    const report = within(reportCard as HTMLElement);

    expect(report.getByText('Stock Out')).toBeInTheDocument();
    expect(report.getByText('#17')).toBeInTheDocument();
    expect(report.getByText('Detail item per transaksi')).toBeInTheDocument();
    expect(report.getByText('Riwayat proses transaksi')).toBeInTheDocument();
    expect(report.getByText('SKU-001')).toBeInTheDocument();
    expect(report.getByText('Acme Corp')).toBeInTheDocument();
    expect(report.getByText('Delivered to customer')).toBeInTheDocument();
  });

  it('opens the browser print dialog from the report action button', () => {
    mockUseGetReportsQuery.mockReturnValue({
      data: {
        items: [
          {
            id: 17,
            type: 'STOCK_OUT',
            status: 'DONE',
            referenceCode: 'OUT-MANUAL-002',
            note: 'Pengiriman final',
            completedAt: '2026-04-01T12:15:00Z',
            createdAt: '2026-04-01T09:00:00Z',
            updatedAt: '2026-04-01T12:15:00Z',
            items: [
              {
                inventoryId: 1,
                sku: 'SKU-001',
                name: 'Widget A',
                customerName: 'Acme Corp',
                quantity: 20,
              },
            ],
            history: [],
          },
        ],
        total: 1,
        limit: 10,
        offset: 0,
        unitsIn: 0,
        unitsOut: 20,
      },
      isLoading: false,
      isFetching: false,
      error: undefined,
    });

    render(<ReportsPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Print report' }));

    expect(window.print).toHaveBeenCalledTimes(1);
  });

  it('applies report filters to the paginated query and exports all matching filtered reports', async () => {
    mockUseGetReportsQuery.mockReturnValue({
      data: {
        items: [
          {
            id: 17,
            type: 'STOCK_OUT',
            status: 'DONE',
            referenceCode: 'OUT-MANUAL-002',
            note: 'Pengiriman final',
            completedAt: '2026-04-01T12:15:00Z',
            createdAt: '2026-04-01T09:00:00Z',
            updatedAt: '2026-04-01T12:15:00Z',
            items: [
              {
                inventoryId: 1,
                sku: 'SKU-001',
                name: 'Widget A',
                customerName: 'Acme Corp',
                quantity: 20,
              },
            ],
            history: [],
          },
        ],
        total: 1,
        limit: 10,
        offset: 0,
        unitsIn: 0,
        unitsOut: 20,
      },
      isLoading: false,
      isFetching: false,
      error: undefined,
    });
    mockFetch.mockResolvedValue({
      ok: true,
      blob: () => Promise.resolve(new Blob(['csv-data'], { type: 'text/csv' })),
      headers: {
        get: (name: string) => (name === 'Content-Disposition' ? 'attachment; filename="filtered-report.csv"' : null),
      },
    });

    render(<ReportsPage />);

    fireEvent.change(screen.getByLabelText('Type'), { target: { value: 'STOCK_OUT' } });
    fireEvent.change(screen.getByPlaceholderText('OUT-MANUAL'), { target: { value: 'OUT-MANUAL' } });
    fireEvent.change(screen.getByLabelText('Completed from'), { target: { value: '2026-04-01' } });
    fireEvent.change(screen.getByLabelText('Completed to'), { target: { value: '2026-04-02' } });

    fireEvent.click(screen.getByRole('button', { name: 'Apply filters' }));

    expect(mockUseGetReportsQuery).toHaveBeenLastCalledWith({
      type: 'STOCK_OUT',
      referenceCode: 'OUT-MANUAL',
      completedFrom: '2026-04-01',
      completedTo: '2026-04-02',
      limit: 10,
      offset: 0,
    });

    fireEvent.click(screen.getByRole('button', { name: 'Export CSV' }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    expect(mockFetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/reports/export?type=STOCK_OUT&referenceCode=OUT-MANUAL&completedFrom=2026-04-01&completedTo=2026-04-02',
      {
        headers: {
          Accept: 'text/csv',
        },
      },
    );

    await waitFor(() => {
      expect(screen.getByText('Filtered report export completed.')).toBeInTheDocument();
    });
  });
});