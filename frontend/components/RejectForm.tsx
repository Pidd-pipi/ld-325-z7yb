'use client';

import { useState } from 'react';

import { Button } from '@/components/ui/button';

interface RejectFormProps {
  onSubmit: (reason: string) => Promise<void>;
  onCancel: () => void;
}

export function RejectForm({ onSubmit, onCancel }: RejectFormProps) {
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const submit = async () => {
    if (reason.trim().length < 2) {
      setError('请填写无法供货的原因，将随采购单保留');
      return;
    }
    setPending(true);
    setError('');
    try {
      await onSubmit(reason.trim());
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交失败，请稍后再试');
    } finally {
      setPending(false);
    }
  };
  return (
    <div className="order-form">
      <label className="grow">
        无法供货原因（随单保留）
        <input value={reason} maxLength={200} placeholder="如：厂家排产已满，本月无法供货" onChange={(event) => setReason(event.target.value)} />
      </label>
      <Button onClick={submit} disabled={pending}>{pending ? '提交中…' : '确认无法供货'}</Button>
      <button className="link" onClick={onCancel}>返回</button>
      {error && <p className="form-error">{error}</p>}
    </div>
  );
}
