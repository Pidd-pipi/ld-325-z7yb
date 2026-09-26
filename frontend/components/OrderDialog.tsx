'use client';

import { useEffect, useState } from 'react';
import { X } from 'lucide-react';

import type { Offer, Product } from '@/lib/types';
import { cnDate, money } from '@/lib/utils';
import { Button } from '@/components/ui/button';

interface OrderDialogProps {
  product: Product;
  offer: Offer;
  onClose: () => void;
  onSubmit: (quantity: number) => Promise<void>;
}

export function OrderDialog({ product, offer, onClose, onSubmit }: OrderDialogProps) {
  const [quantity, setQuantity] = useState(offer.MOQ);
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const expected = new Date(Date.now() + offer.DeliveryDays * 86400000);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const submit = async () => {
    if (quantity < offer.MOQ) {
      setError(`起订量为 ${offer.MOQ} ${product.Unit}，请调整下单数量`);
      return;
    }
    setPending(true);
    setError('');
    try {
      await onSubmit(quantity);
    } catch (err) {
      setError(err instanceof Error ? err.message : '下单失败，请稍后重试');
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div className="dialog" role="dialog" aria-modal="true" aria-label="确认采购单" onClick={(event) => event.stopPropagation()}>
        <header>
          <b>确认采购单</b>
          <button className="icon-button" onClick={onClose} aria-label="关闭"><X size={16} /></button>
        </header>
        <p className="dialog-product">{product.Name} · {offer.Supplier.Name}</p>
        <dl className="dialog-terms">
          <div><dt>锁定单价</dt><dd>{money(offer.UnitPrice)} / {product.Unit}</dd></div>
          <div><dt>运费说明</dt><dd>{offer.Freight}</dd></div>
          <div><dt>预计到货</dt><dd>{cnDate(expected.toISOString())} · {offer.DeliveryDays} 天</dd></div>
        </dl>
        <label className="dialog-field">
          下单数量（起订 {offer.MOQ} {product.Unit}）
          <input type="number" min={offer.MOQ} value={quantity} onChange={(event) => setQuantity(Number(event.target.value))} />
        </label>
        <div className="dialog-total"><span>货款合计</span><strong>{money(quantity * offer.UnitPrice)}</strong></div>
        {error && <p className="dialog-error" role="alert">{error}</p>}
        <Button onClick={submit} disabled={pending}>{pending ? '提交中…' : '确认下单'}</Button>
        <p className="dialog-note">下单后商家、单价、运费与预计到货日即刻锁定；商家接单前可取消，接单后按实际批次登记到货。</p>
      </div>
    </div>
  );
}
