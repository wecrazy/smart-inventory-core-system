import { FormEvent, useState } from 'react';

import {
  useCancelStockInMutation,
  useCreateStockInMutation,
  useGetInventoryQuery,
  useGetStockInQuery,
  useUpdateStockInStatusMutation,
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

export function StockInPage() {
  const [lines, setLines] = useState<DraftLine[]>([{ inventoryId: '', quantity: '1' }]);
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const { data: inventory = [] } = useGetInventoryQuery();
  const { data: transactions = [], isLoading } = useGetStockInQuery();
  const [createStockIn, { isLoading: isCreating }] = useCreateStockInMutation();
  const [updateStatus, { isLoading: isUpdating }] = useUpdateStockInStatusMutation();
  const [cancelStockIn, { isLoading: isCancelling }] = useCancelStockInMutation();

  const hasInventory = inventory.length > 0;
  const canSubmit = hasInventory && lines.every((line) => line.inventoryId !== '' && Number(line.quantity) > 0);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const formData = new FormData(form);

    try {
      await createStockIn({
        referenceCode: String(formData.get('referenceCode') ?? ''),
        note: String(formData.get('note') ?? ''),
        items: lines.map((line) => ({
          inventoryId: Number(line.inventoryId),
          quantity: Number(line.quantity),
        })),
      }).unwrap();

      form.reset();
      setLines([{ inventoryId: '', quantity: '1' }]);
      setFeedback({ tone: 'success', message: 'Stock-in transaction created in CREATED state.' });
    } catch (submitError) {
      setFeedback({
        tone: 'error',
        message: extractApiErrorMessage(submitError, 'Failed to create stock-in transaction.'),
      });
    }
  }

  async function handlePromote(transaction: Transaction) {
    const nextStatus = transaction.status === 'CREATED' ? 'IN_PROGRESS' : 'DONE';

    try {
      await updateStatus({ id: transaction.id, status: nextStatus }).unwrap();
      setFeedback({ tone: 'success', message: `Stock-in ${transaction.referenceCode} moved to ${nextStatus}.` });
    } catch (submitError) {
      setFeedback({
        tone: 'error',
        message: extractApiErrorMessage(submitError, `Failed to update stock-in ${transaction.referenceCode}.`),
      });
    }
  }

  async function handleCancel(transaction: Transaction) {
    try {
      await cancelStockIn({ id: transaction.id }).unwrap();
      setFeedback({ tone: 'success', message: `Stock-in ${transaction.referenceCode} cancelled.` });
    } catch (submitError) {
      setFeedback({
        tone: 'error',
        message: extractApiErrorMessage(submitError, `Failed to cancel stock-in ${transaction.referenceCode}.`),
      });
    }
  }

  return (
    <div className="stack page-grid">
      <section className="panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Inbound workflow</p>
            <h2>Create stock-in transaction</h2>
          </div>
        </div>
        <p className="helper-copy">Stock fisik inventory baru bertambah setelah transaksi stock-in dipindahkan sampai status DONE.</p>
        {feedback ? <FeedbackBanner message={feedback.message} onClose={() => setFeedback(null)} tone={feedback.tone} /> : null}
        {!hasInventory ? (
          <p className="helper-copy">
            Belum ada inventory yang bisa dipakai. Buka halaman Inventory dan buat minimal satu item terlebih dahulu.
          </p>
        ) : null}
        <form className="stack" onSubmit={handleSubmit}>
          <input className="field" name="referenceCode" placeholder="Optional reference code" />
          <textarea className="field textarea" name="note" placeholder="Inbound note" rows={4} />
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
                      {item.sku} · {item.name}
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
            {isCreating ? 'Creating…' : 'Create stock-in'}
          </button>
        </form>
      </section>

      <section className="panel panel-wide">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Inbound queue</p>
            <h2>Status progression</h2>
          </div>
        </div>
        {isLoading ? <p>Loading stock-in transactions…</p> : null}
        <div className="stack">
          {!isLoading && transactions.length === 0 ? (
            <div className="empty-state">Belum ada transaksi stock-in. Buat transaksi baru dari panel kiri untuk mulai menguji workflow CREATED → IN_PROGRESS → DONE.</div>
          ) : null}
          {transactions.map((transaction) => (
            <TransactionCard
              key={transaction.id}
              transaction={transaction}
              onCancel={() => handleCancel(transaction)}
              onPromote={() => handlePromote(transaction)}
              promoteLabel={transaction.status === 'CREATED' ? 'Move to in progress' : 'Mark done'}
              showPromote={transaction.status === 'CREATED' || transaction.status === 'IN_PROGRESS'}
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
            Cancel
          </button>
        ) : null}
      </div>
    </article>
  );
}