'use client';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ArrowDownRight, LoaderCircle, ShieldCheck } from 'lucide-react';
import { api } from '@/lib/api';
import type { Product, PurchaseOrder, Trend } from '@/lib/types';
import { AppHeader } from '@/components/AppHeader';
import { CategoryRail } from '@/components/CategoryRail';
import { ProductCatalog } from '@/components/ProductCatalog';
import { ComparisonTray } from '@/components/ComparisonTray';
import { InsightPanel } from '@/components/InsightPanel';
import { OrderBoard } from '@/components/OrderBoard';
import { BudgetCalculator } from '@/components/BudgetCalculator';
export default function Home() {
  const [products, setProducts] = useState<Product[]>([]);
  const [orders, setOrders] = useState<PurchaseOrder[]>([]);
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('全部');
  const [compareIDs, setCompareIDs] = useState<number[]>([]);
  const [selected, setSelected] = useState<Product | null>(null);
  const [trend, setTrend] = useState<Trend | null>(null);
  const [trendRange, setTrendRange] = useState<'30d' | '90d' | '1y'>('30d');
  const [sort, setSort] = useState<'price' | 'sales' | 'rating'>('rating');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(true);
  const notify = (text: string) => { setMessage(text); window.setTimeout(() => setMessage(''), 2800); };
  const refreshOrders = useCallback(() => { api.listOrders().then(setOrders).catch(() => {}); }, []);
  useEffect(() => { api.listProducts().then((data) => { setProducts(data.items); setSelected(data.items[0] || null); }).catch(() => setMessage('报价数据暂时不可用，请确认后端服务已启动。')).finally(() => setLoading(false)); refreshOrders(); }, [refreshOrders]);
  useEffect(() => { if (selected) api.trend(selected.ID, trendRange).then(setTrend).catch(() => setTrend(null)); }, [selected, trendRange]);
  const visible = useMemo(() => products.filter((item) => (category === '全部' || item.Category?.Name === category) && `${item.Name}${item.Brand}${item.Model}`.toLowerCase().includes(query.toLowerCase())).sort((a, b) => sort === 'price' ? Math.min(...a.Offers.map((offer) => offer.UnitPrice)) - Math.min(...b.Offers.map((offer) => offer.UnitPrice)) : sort === 'sales' ? b.SalesCount - a.SalesCount : b.Rating - a.Rating), [products, category, query, sort]);
  const compare = (id: number) => setCompareIDs((current) => current.includes(id) ? current.filter((value) => value !== id) : current.length < 4 ? [...current, id] : current);
  const placeOrder = async (offerId: number, quantity: number) => {
    await api.createOrder(offerId, quantity);
    notify('采购单已提交，等待商家接单');
    refreshOrders();
  };
  const runOrderAction = async (action: () => Promise<unknown>, ok: string) => {
    try {
      await action();
      notify(ok);
    } catch (error) {
      notify(error instanceof Error ? error.message : '操作失败，请稍后再试');
    } finally {
      refreshOrders();
    }
  };
  return <main id="top"><AppHeader query={query} onQuery={setQuery}/><section className="hero"><div><p className="eyebrow">MATERIAL MARKET INTELLIGENCE / SINCE 2026</p><h1>不是找最低价。<br/><em>是买到恰好的那一笔。</em></h1><p className="hero-copy">把品牌、规格、交期与多商家报价放到同一张桌子上。今天的采购，应该有据可依。</p><div className="hero-note"><ShieldCheck size={18}/><span>演示数据每日价格记录 · 30 天历史趋势 · 真实可比较字段</span></div></div><div className="hero-number"><span>本期已收录</span><strong>8,624</strong><b>条有效报价 <ArrowDownRight size={18}/></b><p>覆盖瓷砖、地板、涂料、卫浴等<br/>装修决策中的高频材料。</p></div></section><CategoryRail selected={category} onSelected={setCategory}/>{loading ? <div className="loading"><LoaderCircle className="spin"/> 正在汇总市场报价…</div> : <><ProductCatalog products={visible} compareIDs={compareIDs} onCompare={compare} onFavorite={async (id) => { await api.favorite(id); notify('已收入「本周采购」收藏夹'); }} onSelect={setSelected} sort={sort} onSort={setSort}/><ComparisonTray items={products.filter((item) => compareIDs.includes(item.ID))} onRemove={compare}/><InsightPanel product={selected} trend={trend} range={trendRange} onRange={setTrendRange} onAlert={async (id, target) => { await api.alert(id,target); notify('价格预警已建立，降价时会在站内通知'); }} onOrder={placeOrder}/><OrderBoard orders={orders} onAccept={(id) => runOrderAction(() => api.acceptOrder(id), '商家已接单，订单进入在途')} onReject={(id, reason) => runOrderAction(() => api.rejectOrder(id, reason), '商家已说明无法供货，原因已保留')} onCancel={(id) => runOrderAction(() => api.cancelOrder(id), '采购单已取消')} onReceive={(id, quantity, note) => runOrderAction(() => api.receiveOrder(id, quantity, note), '到货已登记')}/><BudgetCalculator onSubmit={async (room,area) => { const result = await api.budget(room,area); notify('预算已保存，可继续替换为实际报价'); return result.Estimate; }}/></>}{message && <div className="toast" role="status">{message}</div>}<footer><span>筑价 BUILD PRICE INDEX</span><span>报价仅作采购决策参考，请以商家最终合同为准。</span></footer></main>;
}
