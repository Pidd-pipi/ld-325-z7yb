'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { ArrowDownRight, LoaderCircle, ShieldCheck } from 'lucide-react';

import { api } from '@/lib/api';
import type { Offer, Product, PurchaseOrder, Trend } from '@/lib/types';
import { AppHeader } from '@/components/AppHeader';
import { CategoryRail } from '@/components/CategoryRail';
import { ProductCatalog } from '@/components/ProductCatalog';
import { ComparisonTray } from '@/components/ComparisonTray';
import { InsightPanel } from '@/components/InsightPanel';
import { OrderDialog } from '@/components/OrderDialog';
import { PurchaseOrders } from '@/components/PurchaseOrders';
import { BudgetCalculator } from '@/components/BudgetCalculator';

type SortKey = 'price' | 'sales' | 'rating';

export default function Home() {
  const [products, setProducts] = useState<Product[]>([]);
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('全部');
  const [compareIDs, setCompareIDs] = useState<number[]>([]);
  const [selected, setSelected] = useState<Product | null>(null);
  const [trend, setTrend] = useState<Trend | null>(null);
  const [trendRange, setTrendRange] = useState<'30d' | '90d' | '1y'>('30d');
  const [sort, setSort] = useState<SortKey>('rating');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(true);
  const [orders, setOrders] = useState<PurchaseOrder[]>([]);
  const [orderTarget, setOrderTarget] = useState<{ product: Product; offer: Offer } | null>(null);
  const [merchantMode, setMerchantMode] = useState(false);
  const [merchantToken, setMerchantToken] = useState('');

  const notify = (text: string) => {
    setMessage(text);
    window.setTimeout(() => setMessage(''), 2800);
  };
  const errorText = (err: unknown) => (err instanceof Error ? err.message : '操作失败，请稍后重试');

  const loadOrders = useCallback(() => {
    api.orders().then(setOrders).catch(() => setOrders([]));
  }, []);

  useEffect(() => {
    api.listProducts()
      .then((data) => {
        setProducts(data.items);
        setSelected(data.items[0] || null);
      })
      .catch(() => setMessage('报价数据暂时不可用，请确认后端服务已启动。'))
      .finally(() => setLoading(false));
    loadOrders();
  }, [loadOrders]);

  useEffect(() => {
    if (selected) api.trend(selected.ID, trendRange).then(setTrend).catch(() => setTrend(null));
  }, [selected, trendRange]);

  const visible = useMemo(
    () =>
      products
        .filter((item) => (category === '全部' || item.Category?.Name === category) && `${item.Name}${item.Brand}${item.Model}`.toLowerCase().includes(query.toLowerCase()))
        .sort((a, b) =>
          sort === 'price'
            ? Math.min(...a.Offers.map((offer) => offer.UnitPrice)) - Math.min(...b.Offers.map((offer) => offer.UnitPrice))
            : sort === 'sales'
              ? b.SalesCount - a.SalesCount
              : b.Rating - a.Rating,
        ),
    [products, category, query, sort],
  );

  const compare = (id: number) =>
    setCompareIDs((current) => (current.includes(id) ? current.filter((value) => value !== id) : current.length < 4 ? [...current, id] : current));

  const runOrderAction = async (action: () => Promise<unknown>, ok: string) => {
    try {
      await action();
      loadOrders();
      notify(ok);
    } catch (err) {
      notify(errorText(err));
    }
  };

  const submitOrder = async (quantity: number) => {
    if (!orderTarget) return;
    await api.createOrder(orderTarget.offer.ID, quantity);
    setOrderTarget(null);
    loadOrders();
    notify('采购单已创建，等待商家接单');
  };

  const toggleMerchant = async (next: boolean) => {
    if (!next) {
      setMerchantMode(false);
      return;
    }
    try {
      if (!merchantToken) {
        const data = await api.demoToken('supplier');
        setMerchantToken(data.token);
      }
      setMerchantMode(true);
    } catch (err) {
      notify(errorText(err));
    }
  };

  return (
    <main id="top">
      <AppHeader query={query} onQuery={setQuery} />
      <section className="hero">
        <div>
          <p className="eyebrow">MATERIAL MARKET INTELLIGENCE / SINCE 2026</p>
          <h1>不是找最低价。<br /><em>是买到恰好的那一笔。</em></h1>
          <p className="hero-copy">把品牌、规格、交期与多商家报价放到同一张桌子上。今天的采购，应该有据可依。</p>
          <div className="hero-note"><ShieldCheck size={18} /><span>演示数据每日价格记录 · 30 天历史趋势 · 真实可比较字段</span></div>
        </div>
        <div className="hero-number">
          <span>本期已收录</span>
          <strong>8,624</strong>
          <b>条有效报价 <ArrowDownRight size={18} /></b>
          <p>覆盖瓷砖、地板、涂料、卫浴等<br />装修决策中的高频材料。</p>
        </div>
      </section>
      <CategoryRail selected={category} onSelected={setCategory} />
      {loading ? (
        <div className="loading"><LoaderCircle className="spin" /> 正在汇总市场报价…</div>
      ) : (
        <>
          <ProductCatalog
            products={visible}
            compareIDs={compareIDs}
            onCompare={compare}
            onFavorite={async (id) => {
              await api.favorite(id);
              notify('已收入「本周采购」收藏夹');
            }}
            onSelect={setSelected}
            sort={sort}
            onSort={setSort}
          />
          <ComparisonTray items={products.filter((item) => compareIDs.includes(item.ID))} onRemove={compare} />
          <InsightPanel
            product={selected}
            trend={trend}
            range={trendRange}
            onRange={setTrendRange}
            onAlert={async (id, target) => {
              await api.alert(id, target);
              notify('价格预警已建立，降价时会在站内通知');
            }}
            onOrder={(product, offer) => setOrderTarget({ product, offer })}
          />
          <PurchaseOrders
            orders={orders}
            merchantMode={merchantMode}
            onMerchantMode={toggleMerchant}
            onCancel={(id) => runOrderAction(() => api.cancelOrder(id), '采购单已取消')}
            onArrive={(id, quantity, note) => runOrderAction(() => api.registerArrival(id, quantity, note), '到货已登记')}
            onAccept={(id) => runOrderAction(() => api.acceptOrder(id, merchantToken), '商家已接单，采购单进入在途')}
            onReject={(id, reason) => runOrderAction(() => api.rejectOrder(id, reason, merchantToken), '已记录商家无法供货说明')}
          />
          <BudgetCalculator
            onSubmit={async (room, area) => {
              const result = await api.budget(room, area);
              notify('预算已保存，可继续替换为实际报价');
              return result.Estimate;
            }}
          />
        </>
      )}
      {orderTarget && <OrderDialog product={orderTarget.product} offer={orderTarget.offer} onClose={() => setOrderTarget(null)} onSubmit={submitOrder} />}
      {message && <div className="toast" role="status">{message}</div>}
      <footer>
        <span>筑价 BUILD PRICE INDEX</span>
        <span>报价仅作采购决策参考，请以商家最终合同为准。</span>
      </footer>
    </main>
  );
}
