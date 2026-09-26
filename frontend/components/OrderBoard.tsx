'use client';

import { useMemo, useState } from 'react';
import { ClipboardList } from 'lucide-react';

import type { PurchaseOrder } from '@/lib/types';
import { orderStatusLabel, orderTabs } from '@/lib/orderMeta';
import { OrderCard } from './OrderCard';

interface OrderBoardProps {
  orders: PurchaseOrder[];
  onAccept: (id: number) => Promise<void>;
  onReject: (id: number, reason: string) => Promise<void>;
  onCancel: (id: number) => Promise<void>;
  onReceive: (id: number, quantity: number, note: string) => Promise<void>;
}

export function OrderBoard({ orders, ...actions }: OrderBoardProps) {
  const [tab, setTab] = useState('');
  const visible = useMemo(() => (tab ? orders.filter((order) => order.Status === tab) : orders), [orders, tab]);
  const count = (value: string) => (value ? orders.filter((order) => order.Status === value).length : orders.length);
  const emptyLabel = tab ? `暂无「${orderStatusLabel[tab]}」采购单` : '暂无采购单';
  return (
    <section id="orders" className="orders">
      <div className="section-title">
        <div>
          <p className="eyebrow">PURCHASE ORDERS</p>
          <h2>采购单全程可追，<em>到货一笔一笔记。</em></h2>
        </div>
        <p>下单即锁定商家、单价、运费与预计到货日；商家接单后分次登记到货，全部到齐才算完成，逾期单会标注延迟天数。</p>
      </div>
      <div className="order-tabs" role="group" aria-label="采购单状态筛选">
        {orderTabs.map((item) => (
          <button key={item.value} className={tab === item.value ? 'active' : ''} onClick={() => setTab(item.value)}>
            {item.label} · {count(item.value)}
          </button>
        ))}
      </div>
      {visible.length === 0 ? (
        <p className="orders-empty"><ClipboardList size={16} /> {emptyLabel}，去报价旁点击「下单」创建一笔。</p>
      ) : (
        <div className="order-grid">
          {visible.map((order) => <OrderCard key={order.ID} order={order} {...actions} />)}
        </div>
      )}
    </section>
  );
}
