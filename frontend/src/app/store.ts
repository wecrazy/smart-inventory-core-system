import { configureStore } from '@reduxjs/toolkit';

import { inventoryApi } from './api';

export const store = configureStore({
  reducer: {
    [inventoryApi.reducerPath]: inventoryApi.reducer,
  },
  middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(inventoryApi.middleware),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;