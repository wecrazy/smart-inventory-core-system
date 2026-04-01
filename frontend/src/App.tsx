import { NavLink, Outlet } from 'react-router-dom';

const navigation = [
  { label: 'Inventory', to: '/' },
  { label: 'Stock In', to: '/stock-in' },
  { label: 'Stock Out', to: '/stock-out' },
  { label: 'Reports', to: '/reports' },
];

export function App() {
  return (
    <div className="shell">
      <header className="hero">
        <div>
          <p className="eyebrow">Senior Fullstack Assessment</p>
          <h1>Smart Inventory Core System</h1>
          <p className="hero-copy">
            Inventory visibility, auditable stock movement, and reservation-safe outbound handling in one workspace.
          </p>
        </div>
        <nav className="tabs" aria-label="Primary navigation">
          {navigation.map((item) => (
            <NavLink
              className={({ isActive }) => (isActive ? 'tab tab-active' : 'tab')}
              key={item.to}
              to={item.to}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </header>
      <main className="page-content">
        <Outlet />
      </main>
    </div>
  );
}

export default App;