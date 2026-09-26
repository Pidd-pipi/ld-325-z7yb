'use client';

import { useState } from 'react';
import { CalendarClock, CheckCircle2, PackageCheck, Truck, XCircle } from 'lucide-react';

import type { PurchaseOrder } from '@/lib/types';
import { orderStatusLabel, orderStatusTone } from '@/lib/orderMeta';
import { cnDate, money } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { DeliveryForm } from './DeliveryForm';
import { RejectForm } from './RejectForm';

interface OrderCardProps {
  order: PurchaseOrder;
  onAccept: (id: number) => Promise<void>;
  onReject: (id: number, reason: string) => Promise<void>;
  onCancel: (id: number) => Promise<void>;
  onReceive: (id: number, quantity: number, note: string) => Promise<void>;
}

export function OrderCard({ order, onAccept, onReject, onCancel, onReceive }: OrderCardProps) {
  const [receiving, setReceiving] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  const unit = order.Product?.Unit || '';
  const remaining = order.Quantity - order.ReceivedTotal;
  const progress = order.Quantity > 0 ? Math.min(100, Math.round((order.ReceivedTotal / order.Quantity) * 100)) : 0;
  const awaiting = order.Status === 'pending';
  const shipping = order.Status === 'in_transit' || order.Status === 'partial';
  return (
    <article className="order-card">
      <div className="order-head">
        <div>
          <b>{order.OrderNo}</b>
          <span>{cnDate(order.CreatedAt)} 下单 · {order.Supplier?.Name}</span>
        </div>
        <div className="order-badges">
          <Badge tone={orderStatusTone[order.Status] || 'neutral'}>{orderStatusLabel[order.Status] || order.Status}</Badge>
          {order.DelayDays > 0 && <Badge tone="alert">已逾期 {order.DelayDays} 天</Badge>}
        </div>
      </div>
      <h3>{order.Product?.Name}</h3>
      <p className="order-sub">{order.Product?.Brand} · {order.Product?.Model} · 下单 {order.Quantity} {unit}</p>
      <dl className="order-terms">
        <div><dt>锁定单价</dt><dd>{money(order.UnitPrice)}</dd></div>
        <div><dt>运费说明</dt><dd>{order.Freight}</dd></div>
        <div><dt>预计到货</dt><dd>{cnDate(order.ExpectedArrival)}</dd></div>
        <div><dt>合计金额</dt><dd>{money(order.TotalAmount)}</dd></div>
      </dl>
      {(shipping || order.Status === 'completed') && (
        <div className="order-progress">
          <div className="order-progress-bar"><span style={{ width: `${progress}%` }} /></div>
          <span>已到货 {order.ReceivedTotal} / {order.Quantity} {unit}</span>
        </div>
      )}
      {order.Deliveries && order.Deliveries.length > 0 && (
        <ul className="order-deliveries">
          {order.Deliveries.map((delivery) => (
            <li key={delivery.ID}><PackageCheck size={13} /> {cnDate(delivery.CreatedAt)} 到货 {delivery.Quantity} {unit}{delivery.Note ? ` · ${delivery.Note}` : ''}</li>
          ))}
        </ul>
      )}
      {order.Status === 'rejected' && <p className="order-reason"><XCircle size={13} /> 商家无法供货：{order.RejectReason}</p>}
      {order.Status === 'completed' && order.CompletedAt && <p className="order-done"><CheckCircle2 size={13} /> {cnDate(order.CompletedAt)} 全部到货，采购完成</p>}
      {awaiting && (
        <div className="order-actions">
          <Button onClick={() => onAccept(order.ID)}><Truck size={14} /> 商家接单</Button>
          <button className="link" onClick={() => setRejecting(!rejecting)}>无法供货</button>
          <button className="link danger" onClick={() => onCancel(order.ID)}>取消订单</button>
        </div>
      )}
      {shipping && (
        <div className="order-actions">
          <Button onClick={() => setReceiving(!receiving)}><PackageCheck size={14} /> 登记到货（余 {remaining} {unit}）</Button>
        </div>
      )}
      {rejecting && <RejectForm onSubmit={async (reason) => { await onReject(order.ID, reason); setRejecting(false); }} onCancel={() => setRejecting(false)} />}
      {receiving && <DeliveryForm remaining={remaining} unit={unit} onSubmit={async (quantity, note) => { await onReceive(order.ID, quantity, note); setReceiving(false); }} onCancel={() => setReceiving(false)} />}
      {awaiting && <p className="order-hint"><CalendarClock size={13} /> 等待商家接单，接单前可取消；演示环境可代商家接单或填写无法供货原因。</p>}
    </article>
  );
}
