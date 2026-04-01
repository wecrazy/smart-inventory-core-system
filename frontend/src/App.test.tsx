import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

import App from './App';

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');

  return {
    ...actual,
    Outlet: () => <div>Inventory content</div>,
  };
});

describe('App shell', () => {
  it('renders the primary navigation and outlet content', () => {
    render(
      <MemoryRouter
        initialEntries={['/']}
        future={{
          v7_startTransition: true,
          v7_relativeSplatPath: true,
        }}
      >
        <App />
      </MemoryRouter>,
    );

    expect(screen.getByRole('heading', { name: /smart inventory core system/i })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Inventory' })).toHaveClass('tab-active');
    expect(screen.getByRole('link', { name: 'Stock In' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Stock Out' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Reports' })).toBeInTheDocument();
    expect(screen.getByText('Inventory content')).toBeInTheDocument();
  });
});