'use client';

import { useMemo, useState } from 'react';
import { Store, User } from 'lucide-react';

import type { OrderStatus, PurchaseOrder } from '@/lib/types';
import { OrderCard, orderStatusMeta } from './OrderCard';

type OrderTab = 'all' | Extract<OrderStatus, 'pending' | 'in_transit' | 'partial' | 'completed'>;

const tabs: OrderTab[] = ['all', 'pending', 'in_transit', 'partial', 'completed'];

interface PurchaseOrdersProps {
  orders: PurchaseOrder[];
  merchantMode: boolean;
  onMerchantMode: (next: boolean) => Promise<void>;
  onCancel: (id: number) => Promise<void>;
  onArrive: (id: number, quantity: number, note: string) => Promise<void>;
  onAccept: (id: number) => Promise<void>;
  onReject: (id: number, reason: string) => Promise<void>;
}

export function PurchaseOrders({ orders, merchantMode, onMerchantMode, onCancel, onArrive, onAccept, onReject }: PurchaseOrdersProps) {
  const [tab, setTab] = useState<OrderTab>('all');
  const counts = useMemo(() => {
    const tally: Record<OrderTab, number> = { all: orders.length, pending: 0, in_transit: 0, partial: 0, completed: 0 };
    orders.forEach((order) => {
      if (order.Status in tally) tally[order.Status as OrderTab] += 1;
    });
    return tally;
  }, [orders]);
  const visible = tab === 'all' ? orders : orders.filter((order) => order.Status === tab);

  return (
    <section id="orders" className="orders">
      <div className="section-title">
        <div>
          <p className="eyebrow">PURCHASE PIPELINE</p>
          <h2>采购单，<em>从下单到到货都有凭据。</em></h2>
        </div>
        <div className="orders-side">
          <p>下单即锁定商家、单价、运费与预计到货日；商家接单后分次登记到货，逾期自动标注延迟天数。</p>
          <button className={merchantMode ? 'mode-toggle active' : 'mode-toggle'} onClick={() => onMerchantMode(!merchantMode)}>
            {merchantMode ? <User size={14} /> : <Store size={14} />}
            {merchantMode ? '返回采购员视角' : '切换到商家视角'}
          </button>
        </div>
      </div>
      {merchantMode && <p className="merchant-note">商家视角：对「待接单」采购单执行接单，或说明无法供货并保留原因（演示环境以 demo-supplier 身份操作）。</p>}
      <div className="order-tabs" role="tablist" aria-label="采购单状态筛选">
        {tabs.map((value) => (
          <button key={value} role="tab" aria-selected={tab === value} className={tab === value ? 'active' : ''} onClick={() => setTab(value)}>
            {value === 'all' ? '全部' : orderStatusMeta[value].label} · {counts[value]}
          </button>
        ))}
      </div>
      {visible.length === 0 ? (
        <p className="orders-empty">{orders.length === 0 ? '还没有采购单。在材料报价旁点击「下单」创建第一笔。' : '该状态下暂无采购单。'}</p>
      ) : (
        <div className="order-grid">
          {visible.map((order) => (
            <OrderCard key={order.ID} order={order} merchantMode={merchantMode} onCancel={onCancel} onArrive={onArrive} onAccept={onAccept} onReject={onReject} />
          ))}
        </div>
      )}
    </section>
  );
}
