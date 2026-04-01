import { FormEvent, startTransition, useDeferredValue, useState } from 'react';

import {
  useAdjustInventoryMutation,
  useCreateInventoryMutation,
  useGetInventoryQuery,
} from '../../app/api';
import { FeedbackBanner } from '../../components/FeedbackBanner';
import { extractApiErrorMessage } from '../../app/errorMessage';

type FeedbackState = {
  tone: 'success' | 'error';
  message: string;
};

export function InventoryPage() {
  const [search, setSearch] = useState('');
  const [sku, setSku] = useState('');
  const [customer, setCustomer] = useState('');
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const [createInventory, { isLoading: isCreating }] = useCreateInventoryMutation();
  const [adjustInventory, { isLoading: isAdjusting }] = useAdjustInventoryMutation();

  const deferredSearch = useDeferredValue(search);
  const deferredSku = useDeferredValue(sku);
  const deferredCustomer = useDeferredValue(customer);

  const { data: inventory = [], isLoading, error } = useGetInventoryQuery({
    search: deferredSearch,
    sku: deferredSku,
    customer: deferredCustomer,
  });
  const { data: adjustmentInventory = [] } = useGetInventoryQuery();

  const hasFilters = deferredSearch.trim() !== '' || deferredSku.trim() !== '' || deferredCustomer.trim() !== '';

  async function handleCreateInventory(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const formData = new FormData(form);

    try {
      await createInventory({
        sku: String(formData.get('sku') ?? ''),
        name: String(formData.get('name') ?? ''),
        customerName: String(formData.get('customerName') ?? ''),
        physicalStock: Number(formData.get('physicalStock') ?? 0),
      }).unwrap();

      form.reset();
      setFeedback({ tone: 'success', message: 'Inventory item created successfully.' });
    } catch (submitError) {
      setFeedback({
        tone: 'error',
        message: extractApiErrorMessage(submitError, 'Failed to create inventory item.'),
      });
    }
  }

  async function handleAdjustment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const formData = new FormData(form);

    try {
      await adjustInventory({
        inventoryId: Number(formData.get('inventoryId') ?? 0),
        newPhysicalStock: Number(formData.get('newPhysicalStock') ?? 0),
        referenceCode: String(formData.get('referenceCode') ?? ''),
        note: String(formData.get('note') ?? ''),
      }).unwrap();

      form.reset();
      setFeedback({ tone: 'success', message: 'Stock adjustment applied successfully.' });
    } catch (submitError) {
      setFeedback({
        tone: 'error',
        message: extractApiErrorMessage(submitError, 'Failed to apply stock adjustment.'),
      });
    }
  }

  return (
    <div className="stack page-grid">
      <section className="panel panel-wide">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Inventory visibility</p>
            <h2>Physical vs available stock</h2>
          </div>
          <div className="filter-row">
            <input
              className="field"
              placeholder="Search name or SKU"
              value={search}
              onChange={(event) => {
                const value = event.target.value;
                startTransition(() => setSearch(value));
              }}
            />
            <input
              className="field"
              placeholder="SKU"
              value={sku}
              onChange={(event) => {
                const value = event.target.value;
                startTransition(() => setSku(value));
              }}
            />
            <input
              className="field"
              placeholder="Customer"
              value={customer}
              onChange={(event) => {
                const value = event.target.value;
                startTransition(() => setCustomer(value));
              }}
            />
          </div>
        </div>
        <p className="helper-copy">
          Gunakan filter untuk tabel inventory. Form stock adjustment tetap mengambil daftar inventory penuh agar workflow tidak hilang saat filter aktif.
        </p>
        {feedback ? <FeedbackBanner message={feedback.message} onClose={() => setFeedback(null)} tone={feedback.tone} /> : null}
        {isLoading ? <p>Loading inventory…</p> : null}
        {error ? <p className="error-copy">Failed to load inventory.</p> : null}
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>SKU</th>
                <th>Name</th>
                <th>Customer</th>
                <th>Physical</th>
                <th>Reserved</th>
                <th>Available</th>
              </tr>
            </thead>
            <tbody>
              {!isLoading && !error && inventory.length === 0 ? (
                <tr>
                  <td colSpan={6}>
                    <div className="empty-state">
                      {hasFilters
                        ? 'Tidak ada inventory yang cocok dengan filter saat ini. Kosongkan filter untuk melihat seluruh stok.'
                        : 'Belum ada inventory. Buat item pertama terlebih dahulu agar stok fisik dan stok tersedia bisa dipantau di sini.'}
                    </div>
                  </td>
                </tr>
              ) : null}
              {inventory.map((item) => (
                <tr key={item.id}>
                  <td>{item.sku}</td>
                  <td>{item.name}</td>
                  <td>{item.customerName}</td>
                  <td>{item.physicalStock}</td>
                  <td>{item.reservedStock}</td>
                  <td>{item.availableStock}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Inventory master</p>
            <h2>Add an item</h2>
          </div>
        </div>
        <form className="stack" onSubmit={handleCreateInventory}>
          <input className="field" name="sku" placeholder="SKU" required />
          <input className="field" name="name" placeholder="Item name" required />
          <input className="field" name="customerName" placeholder="Customer" required />
          <input className="field" min="0" name="physicalStock" placeholder="Initial physical stock" type="number" required />
          <button className="button" disabled={isCreating} type="submit">
            {isCreating ? 'Saving…' : 'Create inventory item'}
          </button>
        </form>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Auditable change</p>
            <h2>Stock adjustment</h2>
          </div>
        </div>
        {adjustmentInventory.length === 0 ? (
          <p className="helper-copy">
            Stock adjustment membutuhkan minimal satu item inventory. Tambahkan inventory baru dari panel sebelumnya terlebih dahulu.
          </p>
        ) : null}
        <form className="stack" onSubmit={handleAdjustment}>
          <select className="field" name="inventoryId" required>
            <option value="">Choose inventory item</option>
            {adjustmentInventory.map((item) => (
              <option key={item.id} value={item.id}>
                {item.sku} · {item.name}
              </option>
            ))}
          </select>
          <input className="field" min="0" name="newPhysicalStock" placeholder="New physical stock" type="number" required />
          <input className="field" name="referenceCode" placeholder="Optional reference code" />
          <textarea className="field textarea" name="note" placeholder="Adjustment note" rows={4} />
          <button className="button button-secondary" disabled={isAdjusting || adjustmentInventory.length === 0} type="submit">
            {isAdjusting ? 'Applying…' : 'Apply adjustment'}
          </button>
        </form>
      </section>
    </div>
  );
}