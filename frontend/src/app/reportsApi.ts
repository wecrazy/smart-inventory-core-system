import type { ReportPage } from '../types';

import type { ReportFilters } from './apiTypes';
import { unwrap } from './apiTypes';
import { baseApi } from './baseApi';

const reportsEndpoints = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getReports: builder.query<ReportPage, ReportFilters | void>({
      query: (params: ReportFilters | void) => ({
        url: '/reports',
        params: params ?? undefined,
      }),
      transformResponse: unwrap<ReportPage>,
      providesTags: ['Reports'],
    }),
  }),
});

export const { useGetReportsQuery } = reportsEndpoints;