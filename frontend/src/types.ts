export type InventoryItem = {
  id: number;
  sku: string;
  name: string;
  customerName: string;
  physicalStock: number;
  reservedStock: number;
  availableStock: number;
  createdAt: string;
  updatedAt: string;
};

export type TransactionStatus = 'CREATED' | 'ALLOCATED' | 'IN_PROGRESS' | 'DONE' | 'CANCELLED';
export type TransactionType = 'STOCK_IN' | 'STOCK_OUT' | 'ADJUSTMENT';

export type HistoryEntry = {
  status: TransactionStatus;
  note: string;
  createdAt: string;
};

export type TransactionItem = {
  inventoryId: number;
  sku: string;
  name: string;
  customerName: string;
  quantity: number;
};

export type Transaction = {
  id: number;
  type: TransactionType;
  status: TransactionStatus;
  referenceCode: string;
  note: string;
  completedAt?: string | null;
  createdAt: string;
  updatedAt: string;
  items: TransactionItem[];
  history: HistoryEntry[];
};

export type ReportPage = {
  items: Transaction[];
  total: number;
  limit: number;
  offset: number;
  unitsIn: number;
  unitsOut: number;
};

export type ApiEnvelope<T> = {
  data: T;
  error?: string;
};