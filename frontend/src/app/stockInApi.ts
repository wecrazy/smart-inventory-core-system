import type { Transaction } from '../types';

import type {
  CancelTransactionPayload,
  CreateTransactionPayload,
  TransactionStatusFilters,
  UpdateTransactionStatusPayload,
} from './apiTypes';
import { unwrap } from './apiTypes';
import { baseApi } from './baseApi';

const stockInEndpoints = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getStockIn: builder.query<Transaction[], TransactionStatusFilters | void>({
      query: (params: TransactionStatusFilters | void) => ({
        url: '/stock-in',
        params: params ?? undefined,
      }),
      transformResponse: unwrap<Transaction[]>,
      providesTags: ['StockIn'],
    }),
    createStockIn: builder.mutation<Transaction, CreateTransactionPayload>({
      query: (body: CreateTransactionPayload) => ({
        url: '/stock-in',
        method: 'POST',
        body,
      }),
      transformResponse: unwrap<Transaction>,
      invalidatesTags: ['StockIn'],
    }),
    updateStockInStatus: builder.mutation<Transaction, UpdateTransactionStatusPayload>({
      query: ({ id, ...body }: UpdateTransactionStatusPayload) => ({
        url: `/stock-in/${id}/status`,
        method: 'PATCH',
        body,
      }),
      transformResponse: unwrap<Transaction>,
      invalidatesTags: ['StockIn', 'Inventory', 'Reports'],
    }),
    cancelStockIn: builder.mutation<Transaction, CancelTransactionPayload>({
      query: ({ id, ...body }: CancelTransactionPayload) => ({
        url: `/stock-in/${id}/cancel`,
        method: 'POST',
        body,
      }),
      transformResponse: unwrap<Transaction>,
      invalidatesTags: ['StockIn', 'Inventory', 'Reports'],
    }),
  }),
});

export const {
  useCancelStockInMutation,
  useCreateStockInMutation,
  useGetStockInQuery,
  useUpdateStockInStatusMutation,
} = stockInEndpoints;