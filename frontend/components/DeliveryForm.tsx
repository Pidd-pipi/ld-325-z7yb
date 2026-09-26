'use client';

import { useState } from 'react';

import { Button } from '@/components/ui/button';

interface DeliveryFormProps {
  remaining: number;
  unit: string;
  onSubmit: (quantity: number, note: string) => Promise<void>;
  onCancel: () => void;
}

export function DeliveryForm({ remaining, unit, onSubmit, onCancel }: DeliveryFormProps) {
  const [quantity, setQuantity] = useState(remaining);
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const submit = async () => {
    if (!Number.isFinite(quantity) || quantity <= 0) {
      setError('到货数量需大于 0');
      return;
    }
    if (quantity > remaining) {
      setError(`累计到货不能超过下单量，本次最多可登记 ${remaining} ${unit}`);
      return;
    }
    setPending(true);
    setError('');
    try {
      await onSubmit(quantity, note.trim());
    } catch (err) {
      setError(err instanceof Error ? err.message : '登记失败，请稍后再试');
    } finally {
      setPending(false);
    }
  };
  return (
    <div className="order-form">
      <label>
        本次到货（{unit}）
        <input type="number" min={1} max={remaining} value={quantity} onChange={(event) => setQuantity(Number(event.target.value))} />
      </label>
      <label className="grow">
        备注（可选）
        <input value={note} maxLength={120} placeholder="如：首批到货，外包装完好" onChange={(event) => setNote(event.target.value)} />
      </label>
      <Button onClick={submit} disabled={pending}>{pending ? '登记中…' : '确认到货'}</Button>
      <button className="link" onClick={onCancel}>取消</button>
      {error && <p className="form-error">{error}</p>}
    </div>
  );
}
