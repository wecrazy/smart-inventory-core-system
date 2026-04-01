import { FormEvent, useState } from 'react';

import {
  useCancelStockOutMutation,
  useCreateStockOutMutation,
  useGetInventoryQuery,
  useGetStockOutQuery,
  useUpdateStockOutStatusMutation,
} from '../../app/api';
import { FeedbackBanner } from '../../components/FeedbackBanner';
import { extractApiErrorMessage } from '../../app/errorMessage';
import type { Transaction } from '../../types';

type DraftLine = {
  inventoryId: string;
  quantity: string;
};

type FeedbackState = {
  tone: 'success' | 'error';
  message: string;
};

export function StockOutPage() {
  const [lines, setLines] = useState<DraftLine[]>([{ inventoryId: '', quantity: '1' }]);
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const { data: inventory = [] } = useGetInventoryQuery();
  const { data: transactions = [], isLoading } = useGetStockOutQuery();
  const [createStockOut, { isLoading: isCreating }] = useCreateStockOutMutation();
  const [updateStatus, { isLoading: isUpdating }] = useUpdateStockOutStatusMutation();
  const [cancelStockOut, { isLoading: isCancelling }] = useCancelStockOutMutation();

  const hasInventory = inventory.length > 0;
  const canSubmit = hasInventory && lines.every((line) => line.inventoryId !== '' && Number(line.quantity) > 0);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const formData = new FormData(form);

    try {
      await createStockOut({
        referenceCode: String(formData.get('referenceCode') ?? ''),
        note: String(formData.get('note') ?? ''),
        items: lines.map((line) => ({
          inventoryId: Number(line.inventoryId),
          quantity: Number(line.quantity),
        })),
      }).unwrap();

      form.reset();
      setLines([{ inventoryId: '', quantity: '1' }]);
      setFeedback({ tone: 'success', message: 'Stock-out transaction created in ALLOCATED state.' });
    } catch (submitError) {
      setFeedback({
        tone: 'error',
        message: extractApiErrorMessage(submitError, 'Failed to allocate stock-out transaction.'),
      });
    }
  }

  async function handlePromote(transaction: Transaction) {
    const nextStatus = transaction.status === 'ALLOCATED' ? 'IN_PROGRESS' : 'DONE';

    try {
      await updateStatus({ id: transaction.id, status: nextStatus }).unwrap();
      setFeedback({ tone: 'success', message: `Stock-out ${transaction.referenceCode} moved to ${nextStatus}.` });
    } catch (submitError) {
      setFeedback({
        tone: 'error',
        message: extractApiErrorMessage(submitError, `Failed to update stock-out ${transaction.referenceCode}.`),
      });
    }
  }

  async function handleCancel(transaction: Transaction) {
    try {
      await cancelStockOut({ id: transaction.id }).unwrap();
      setFeedback({ tone: 'success', message: `Stock-out ${transaction.referenceCode} cancelled and reservation released.` });
    } catch (submitError) {
      setFeedback({
        tone: 'error',
        message: extractApiErrorMessage(submitError, `Failed to cancel stock-out ${transaction.referenceCode}.`),
      });
    }
  }

  return (
    <div className="stack page-grid">
      <section className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Outbound reservation</p>
            <h2>Allocate stock-out</h2>
          </div>
        </div>
        <p className="helper-copy">Saat ALLOCATED, available stock akan berkurang karena stok direservasi. Physical stock baru berkurang ketika status menjadi DONE.</p>
        {feedback ? <FeedbackBanner message={feedback.message} onClose={() => setFeedback(null)} tone={feedback.tone} /> : null}
        {!hasInventory ? (
          <p className="helper-copy">
            Belum ada inventory yang bisa dialokasikan. Tambahkan inventory terlebih dahulu dari halaman Inventory.
          </p>
        ) : null}
        <form className="stack" onSubmit={handleSubmit}>
          <input className="field" name="referenceCode" placeholder="Optional reference code" />
          <textarea className="field textarea" name="note" placeholder="Outbound note" rows={4} />
          <div className="stack">
            {lines.map((line, index) => (
              <div className="inline-grid" key={`${index}-${line.inventoryId}`}>
                <select
                  className="field"
                  required
                  value={line.inventoryId}
                  onChange={(event) => {
                    const nextLines = [...lines];
                    nextLines[index] = { ...nextLines[index], inventoryId: event.target.value };
                    setLines(nextLines);
                  }}
                >
                  <option value="">Choose inventory item</option>
                  {inventory.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.sku} · {item.name} · Available {item.availableStock}
                    </option>
                  ))}
                </select>
                <input
                  className="field"
                  min="1"
                  type="number"
                  value={line.quantity}
                  onChange={(event) => {
                    const nextLines = [...lines];
                    nextLines[index] = { ...nextLines[index], quantity: event.target.value };
                    setLines(nextLines);
                  }}
                />
              </div>
            ))}
          </div>
          <div className="button-row">
            <button className="button button-ghost" disabled={!hasInventory} onClick={() => setLines([...lines, { inventoryId: '', quantity: '1' }])} type="button">
              Add line
            </button>
            {lines.length > 1 ? (
              <button className="button button-ghost" onClick={() => setLines(lines.slice(0, -1))} type="button">
                Remove line
              </button>
            ) : null}
          </div>
          <button className="button" disabled={isCreating || !canSubmit} type="submit">
            {isCreating ? 'Allocating…' : 'Allocate stock-out'}
          </button>
        </form>
      </section>

      <section className="panel panel-wide">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Outbound execution</p>
            <h2>Reservation, packing, completion</h2>
          </div>
        </div>
        {isLoading ? <p>Loading stock-out transactions…</p> : null}
        <div className="stack">
          {!isLoading && transactions.length === 0 ? (
            <div className="empty-state">Belum ada transaksi stock-out. Buat alokasi baru dari panel kiri untuk menguji reservasi, rollback, dan penyelesaian transaksi.</div>
          ) : null}
          {transactions.map((transaction) => (
            <TransactionCard
              key={transaction.id}
              transaction={transaction}
              onCancel={() => handleCancel(transaction)}
              onPromote={() => handlePromote(transaction)}
              promoteLabel={transaction.status === 'ALLOCATED' ? 'Move to in progress' : 'Mark done'}
              showPromote={transaction.status === 'ALLOCATED' || transaction.status === 'IN_PROGRESS'}
              busy={isUpdating || isCancelling}
            />
          ))}
        </div>
      </section>
    </div>
  );
}

function TransactionCard({
  busy,
  onCancel,
  onPromote,
  promoteLabel,
  showPromote,
  transaction,
}: {
  busy: boolean;
  onCancel: () => Promise<unknown>;
  onPromote: () => Promise<unknown>;
  promoteLabel: string;
  showPromote: boolean;
  transaction: Transaction;
}) {
  return (
    <article className="transaction-card">
      <div className="transaction-head">
        <div>
          <h3>{transaction.referenceCode}</h3>
          <p className="muted-copy">{transaction.note || 'No note recorded.'}</p>
        </div>
        <span className={`status-pill status-${transaction.status.toLowerCase()}`}>{transaction.status}</span>
      </div>
      <ul className="item-list">
        {transaction.items.map((item) => (
          <li key={`${transaction.id}-${item.inventoryId}`}>
            {item.sku} · {item.name} · Qty {item.quantity}
          </li>
        ))}
      </ul>
      <div className="button-row">
        {showPromote ? (
          <button className="button" disabled={busy} onClick={() => void onPromote()} type="button">
            {promoteLabel}
          </button>
        ) : null}
        {transaction.status !== 'DONE' && transaction.status !== 'CANCELLED' ? (
          <button className="button button-danger" disabled={busy} onClick={() => void onCancel()} type="button">
            Cancel and rollback
          </button>
        ) : null}
      </div>
    </article>
  );
}