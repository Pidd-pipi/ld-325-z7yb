'use client';

import { useState } from 'react';
import { PackagePlus } from 'lucide-react';

import type { OrderStatus, PurchaseOrder } from '@/lib/types';
import { cnDate, money } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

export const orderStatusMeta: Record<OrderStatus, { label: string; tone: 'good' | 'neutral' | 'alert' }> = {
  pending: { label: '待接单', tone: 'neutral' },
  in_transit: { label: '在途', tone: 'good' },
  partial: { label: '部分到货', tone: 'good' },
  completed: { label: '已完成', tone: 'good' },
  cancelled: { label: '已取消', tone: 'neutral' },
  rejected: { label: '无法供货', tone: 'alert' },
};

interface OrderCardProps {
  order: PurchaseOrder;
  merchantMode: boolean;
  onCancel: (id: number) => Promise<void>;
  onArrive: (id: number, quantity: number, note: string) => Promise<void>;
  onAccept: (id: number) => Promise<void>;
  onReject: (id: number, reason: string) => Promise<void>;
}

export function OrderCard({ order, merchantMode, onCancel, onArrive, onAccept, onReject }: OrderCardProps) {
  const [arrivalOpen, setArrivalOpen] = useState(false);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [quantity, setQuantity] = useState(0);
  const [note, setNote] = useState('');
  const [reason, setReason] = useState('');
  const [pending, setPending] = useState(false);
  const meta = orderStatusMeta[order.Status];
  const remaining = order.Quantity - order.ReceivedQuantity;
  const percent = Math.min(100, Math.round((order.ReceivedQuantity / order.Quantity) * 100));
  const receivable = order.Status === 'in_transit' || order.Status === 'partial';

  const run = async (action: () => Promise<void>) => {
    setPending(true);
    try {
      await action();
    } finally {
      setPending(false);
    }
  };
  const submitArrival = async () => {
    const value = quantity > 0 ? quantity : remaining;
    if (value < 1 || value > remaining) return;
    await run(() => onArrive(order.ID, value, note));
    setArrivalOpen(false);
    setQuantity(0);
    setNote('');
  };
  const submitReject = async () => {
    if (!reason.trim()) return;
    await run(() => onReject(order.ID, reason.trim()));
    setRejectOpen(false);
    setReason('');
  };

  return (
    <article className="order-card">
      <header>
        <div>
          <b>{order.ProductName}</b>
          <span>{order.SupplierName} · 下单于 {cnDate(order.CreatedAt)}</span>
        </div>
        <div className="order-badges">
          <Badge tone={meta.tone}>{meta.label}</Badge>
          {order.DelayDays > 0 && (
            <Badge tone="alert">{order.Status === 'completed' ? `延迟 ${order.DelayDays} 天到货` : `已逾期 ${order.DelayDays} 天`}</Badge>
          )}
        </div>
      </header>
      <dl className="order-terms">
        <div><dt>锁定单价</dt><dd>{money(order.UnitPrice)} / {order.ProductUnit}</dd></div>
        <div><dt>下单量</dt><dd>{order.Quantity} {order.ProductUnit}</dd></div>
        <div><dt>运费</dt><dd>{order.Freight}</dd></div>
        <div><dt>预计到货</dt><dd>{cnDate(order.ExpectedArrival)}</dd></div>
      </dl>
      {order.Status !== 'pending' && order.Status !== 'cancelled' && order.Status !== 'rejected' && (
        <div>
          <div className="order-progress"><span style={{ width: `${percent}%` }} /></div>
          <p className="order-progress-note">已到货 {order.ReceivedQuantity}/{order.Quantity} {order.ProductUnit}</p>
        </div>
      )}
      {order.Arrivals.length > 0 && (
        <ul className="arrival-list">
          {order.Arrivals.map((arrival) => (
            <li key={arrival.ID}>
              <span>{cnDate(arrival.ArrivedAt)} 到货 {arrival.Quantity} {order.ProductUnit}</span>
              {arrival.Note && <span>{arrival.Note}</span>}
            </li>
          ))}
        </ul>
      )}
      {order.RejectReason && <p className="order-reason">商家说明：{order.RejectReason}</p>}
      <footer className="order-actions">
        {!merchantMode && order.Status === 'pending' && (
          <Button disabled={pending} onClick={() => run(() => onCancel(order.ID))}>取消订单</Button>
        )}
        {!merchantMode && receivable && !arrivalOpen && (
          <Button disabled={pending} onClick={() => setArrivalOpen(true)}><PackagePlus size={14} />登记到货</Button>
        )}
        {merchantMode && order.Status === 'pending' && (
          <>
            <Button disabled={pending} onClick={() => run(() => onAccept(order.ID))}>商家接单</Button>
            {!rejectOpen && <Button disabled={pending} onClick={() => setRejectOpen(true)}>无法供货</Button>}
          </>
        )}
      </footer>
      {!merchantMode && arrivalOpen && receivable && (
        <div className="order-form">
          <label>本次到货数量（最多 {remaining}）
            <input type="number" min={1} max={remaining} placeholder={`${remaining}`} value={quantity || ''} onChange={(event) => setQuantity(Number(event.target.value))} />
          </label>
          <label>备注（可选）
            <input value={note} placeholder="批次、验收情况" onChange={(event) => setNote(event.target.value)} />
          </label>
          <Button disabled={pending} onClick={submitArrival}>确认登记</Button>
        </div>
      )}
      {merchantMode && rejectOpen && order.Status === 'pending' && (
        <div className="order-form">
          <label>无法供货原因（将保留在采购记录中）
            <input value={reason} placeholder="如：缺货，预计恢复时间" onChange={(event) => setReason(event.target.value)} />
          </label>
          <Button disabled={pending || !reason.trim()} onClick={submitReject}>提交说明</Button>
        </div>
      )}
    </article>
  );
}
