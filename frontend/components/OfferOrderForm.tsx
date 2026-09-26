'use client';

import { useState } from 'react';
import { Lock } from 'lucide-react';

import type { Offer } from '@/lib/types';
import { money } from '@/lib/utils';
import { Button } from '@/components/ui/button';

interface OfferOrderFormProps {
  offer: Offer;
  unit: string;
  onSubmit: (quantity: number) => Promise<void>;
  onCancel: () => void;
}

export function OfferOrderForm({ offer, unit, onSubmit, onCancel }: OfferOrderFormProps) {
  const [quantity, setQuantity] = useState(offer.MOQ);
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const submit = async () => {
    if (!Number.isFinite(quantity) || quantity < offer.MOQ) {
      setError(`数量不能低于起订量 ${offer.MOQ} ${unit}`);
      return;
    }
    setPending(true);
    setError('');
    try {
      await onSubmit(quantity);
    } catch (err) {
      setError(err instanceof Error ? err.message : '下单失败，请稍后再试');
    } finally {
      setPending(false);
    }
  };
  return (
    <div className="offer-order">
      <label>
        采购数量（{unit}）
        <input type="number" min={offer.MOQ} value={quantity} onChange={(event) => setQuantity(Number(event.target.value))} />
      </label>
      <p className="hint">
        <Lock size={12} /> 下单即锁定：{offer.Supplier.Name} · 单价 {money(offer.UnitPrice)} · {offer.Freight} · 预计 {offer.DeliveryDays} 天到货
      </p>
      <div className="actions">
        <Button onClick={submit} disabled={pending}>{pending ? '提交中…' : '确认下单'}</Button>
        <button className="link" onClick={onCancel}>取消</button>
      </div>
      {error && <p className="form-error">{error}</p>}
    </div>
  );
}
