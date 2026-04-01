import type { InventoryItem, Transaction } from '../types';

import type {
  AdjustInventoryPayload,
  CreateInventoryPayload,
  InventoryFilters,
} from './apiTypes';
import { unwrap } from './apiTypes';
import { baseApi } from './baseApi';

const inventoryEndpoints = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getInventory: builder.query<InventoryItem[], InventoryFilters | void>({
      query: (filters: InventoryFilters | void) => ({
        url: '/inventory',
        params: filters ?? undefined,
      }),
      transformResponse: unwrap<InventoryItem[]>,
      providesTags: ['Inventory'],
    }),
    createInventory: builder.mutation<InventoryItem, CreateInventoryPayload>({
      query: (body: CreateInventoryPayload) => ({
        url: '/inventory',
        method: 'POST',
        body,
      }),
      transformResponse: unwrap<InventoryItem>,
      invalidatesTags: ['Inventory'],
    }),
    adjustInventory: builder.mutation<Transaction, AdjustInventoryPayload>({
      query: (body: AdjustInventoryPayload) => ({
        url: '/inventory/adjustments',
        method: 'POST',
        body,
      }),
      transformResponse: unwrap<Transaction>,
      invalidatesTags: ['Inventory', 'Reports'],
    }),
  }),
});

export const {
  useAdjustInventoryMutation,
  useCreateInventoryMutation,
  useGetInventoryQuery,
} = inventoryEndpoints;