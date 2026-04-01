import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1';

export const baseApi = createApi({
  reducerPath: 'inventoryApi',
  baseQuery: fetchBaseQuery({
    baseUrl: apiBaseUrl,
  }),
  tagTypes: ['Inventory', 'StockIn', 'StockOut', 'Reports'],
  endpoints: () => ({}),
});