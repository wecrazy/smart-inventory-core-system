import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { InventoryPage } from './InventoryPage';

const mockUseGetInventoryQuery = vi.fn();
const mockCreateInventory = vi.fn();
const mockAdjustInventory = vi.fn();

vi.mock('../../app/api', () => ({
  useGetInventoryQuery: (...args: unknown[]) => mockUseGetInventoryQuery(...args),
  useCreateInventoryMutation: () => [mockCreateInventory, { isLoading: false }],
  useAdjustInventoryMutation: () => [mockAdjustInventory, { isLoading: false }],
}));

describe('InventoryPage', () => {
  beforeEach(() => {
    mockUseGetInventoryQuery.mockReset();
    mockCreateInventory.mockReset();
    mockAdjustInventory.mockReset();

    mockUseGetInventoryQuery.mockReturnValue({
      data: [],
      isLoading: false,
      error: undefined,
    });

    mockCreateInventory.mockReturnValue({
      unwrap: () => Promise.resolve({ id: 1 }),
    });

    mockAdjustInventory.mockReturnValue({
      unwrap: () => Promise.resolve({ id: 1 }),
    });
  });

  it('submits create inventory without reading reset from a null event target', async () => {
    render(<InventoryPage />);

    const createButton = screen.getByRole('button', { name: 'Create inventory item' });
    const form = createButton.closest('form');

    expect(form).not.toBeNull();

    const createForm = within(form as HTMLFormElement);

    fireEvent.change(createForm.getByPlaceholderText('SKU'), { target: { value: 'SKU-001' } });
    fireEvent.change(createForm.getByPlaceholderText('Item name'), { target: { value: 'Widget A' } });
    fireEvent.change(createForm.getByPlaceholderText('Customer'), { target: { value: 'Acme Corp' } });
    fireEvent.change(createForm.getByPlaceholderText('Initial physical stock'), { target: { value: '100' } });

    fireEvent.click(createButton);

    await waitFor(() => {
      expect(screen.getByText('Inventory item created successfully.')).toBeInTheDocument();
    });

    expect(screen.queryByText(/cannot read properties of null/i)).not.toBeInTheDocument();
  });

  it('dismisses the feedback banner when the close button is clicked', async () => {
    render(<InventoryPage />);

    const createButton = screen.getByRole('button', { name: 'Create inventory item' });
    const form = createButton.closest('form');

    expect(form).not.toBeNull();

    const createForm = within(form as HTMLFormElement);

    fireEvent.change(createForm.getByPlaceholderText('SKU'), { target: { value: 'SKU-001' } });
    fireEvent.change(createForm.getByPlaceholderText('Item name'), { target: { value: 'Widget A' } });
    fireEvent.change(createForm.getByPlaceholderText('Customer'), { target: { value: 'Acme Corp' } });
    fireEvent.change(createForm.getByPlaceholderText('Initial physical stock'), { target: { value: '100' } });

    fireEvent.click(createButton);

    await waitFor(() => {
      expect(screen.getByText('Inventory item created successfully.')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Dismiss notification' }));

    expect(screen.queryByText('Inventory item created successfully.')).not.toBeInTheDocument();
  });
});