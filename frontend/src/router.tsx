import { createBrowserRouter } from 'react-router-dom';

import App from './App';
import { InventoryPage } from './features/inventory/InventoryPage';
import { ReportsPage } from './features/reports/ReportsPage';
import { StockInPage } from './features/stock-in/StockInPage';
import { StockOutPage } from './features/stock-out/StockOutPage';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <App />,
    children: [
      { index: true, element: <InventoryPage /> },
      { path: '/stock-in', element: <StockInPage /> },
      { path: '/stock-out', element: <StockOutPage /> },
      { path: '/reports', element: <ReportsPage /> },
    ],
  },
]);