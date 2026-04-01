import type { ApiEnvelope } from '../types';

export type InventoryFilters = {
  search?: string;
  sku?: string;
  customer?: string;
};

export type CreateInventoryPayload = {
  sku: string;
  name: string;
  customerName: string;
  physicalStock: number;
};

export type AdjustInventoryPayload = {
  inventoryId: number;
  newPhysicalStock: number;
  referenceCode?: string;
  note?: string;
};

export type TransactionItemPayload = {
  inventoryId: number;
  quantity: number;
};

export type CreateTransactionPayload = {
  referenceCode?: string;
  note?: string;
  items: TransactionItemPayload[];
};

export type UpdateTransactionStatusPayload = {
  id: number;
  status: string;
  note?: string;
};

export type CancelTransactionPayload = {
  id: number;
  note?: string;
};

export type TransactionStatusFilters = {
  status?: string;
};

export type ReportFilters = {
  limit?: number;
  offset?: number;
  type?: 'STOCK_IN' | 'STOCK_OUT';
  referenceCode?: string;
  completedFrom?: string;
  completedTo?: string;
};

export function unwrap<T>(response: ApiEnvelope<T>): T {
  return response.data;
}