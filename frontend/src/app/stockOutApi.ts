import type { Transaction } from '../types';

import type {
  CancelTransactionPayload,
  CreateTransactionPayload,
  TransactionStatusFilters,
  UpdateTransactionStatusPayload,
} from './apiTypes';
import { unwrap } from './apiTypes';
import { baseApi } from './baseApi';

const stockOutEndpoints = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getStockOut: builder.query<Transaction[], TransactionStatusFilters | void>({
      query: (params: TransactionStatusFilters | void) => ({
        url: '/stock-out',
        params: params ?? undefined,
      }),
      transformResponse: unwrap<Transaction[]>,
      providesTags: ['StockOut'],
    }),
    createStockOut: builder.mutation<Transaction, CreateTransactionPayload>({
      query: (body: CreateTransactionPayload) => ({
        url: '/stock-out',
        method: 'POST',
        body,
      }),
      transformResponse: unwrap<Transaction>,
      invalidatesTags: ['StockOut', 'Inventory'],
    }),
    updateStockOutStatus: builder.mutation<Transaction, UpdateTransactionStatusPayload>({
      query: ({ id, ...body }: UpdateTransactionStatusPayload) => ({
        url: `/stock-out/${id}/status`,
        method: 'PATCH',
        body,
      }),
      transformResponse: unwrap<Transaction>,
      invalidatesTags: ['StockOut', 'Inventory', 'Reports'],
    }),
    cancelStockOut: builder.mutation<Transaction, CancelTransactionPayload>({
      query: ({ id, ...body }: CancelTransactionPayload) => ({
        url: `/stock-out/${id}/cancel`,
        method: 'POST',
        body,
      }),
      transformResponse: unwrap<Transaction>,
      invalidatesTags: ['StockOut', 'Inventory', 'Reports'],
    }),
  }),
});

export const {
  useCancelStockOutMutation,
  useCreateStockOutMutation,
  useGetStockOutQuery,
  useUpdateStockOutStatusMutation,
} = stockOutEndpoints;