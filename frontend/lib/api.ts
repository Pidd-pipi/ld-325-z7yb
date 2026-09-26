import type { ApiEnvelope, Product, PurchaseOrder, Trend } from './types';
const API_ROOT = '/api/v1';
async function request<T>(path: string, init?: RequestInit): Promise<T> { const response = await fetch(`${API_ROOT}${path}`, { headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) }, ...init }); const payload = await response.json() as ApiEnvelope<T>; if (!response.ok || payload.code !== 0) throw new Error(payload.message); return payload.data; }
const post = <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body: body === undefined ? undefined : JSON.stringify(body) });
export const api = {
  listProducts: (q = '') => request<{items: Product[]; total: number}>(`/products?sort=rating&q=${encodeURIComponent(q)}`),
  trend: (id: number, range = '30d') => request<Trend>(`/products/${id}/trend?range=${range}`),
  compare: (ids: number[]) => post<Product[]>('/products/compare', { ids }),
  favorite: (productId: number) => post('/favorites', { product_id: productId, folder: '本周采购' }),
  alert: (productId: number, target: number) => post('/alerts', { product_id: productId, target_price: target, drop_percent: 10 }),
  budget: (room: string, area: number) => post<{Estimate: number; Payload: string}>('/budgets', { room_type: room, area }),
  createOrder: (offerId: number, quantity: number) => post<PurchaseOrder>('/orders', { offer_id: offerId, quantity }),
  listOrders: () => request<PurchaseOrder[]>('/orders'),
  acceptOrder: (id: number) => post<PurchaseOrder>(`/orders/${id}/accept`),
  rejectOrder: (id: number, reason: string) => post<PurchaseOrder>(`/orders/${id}/reject`, { reason }),
  cancelOrder: (id: number) => post<PurchaseOrder>(`/orders/${id}/cancel`),
  receiveOrder: (id: number, quantity: number, note: string) => post<PurchaseOrder>(`/orders/${id}/deliveries`, { quantity, note }),
};
