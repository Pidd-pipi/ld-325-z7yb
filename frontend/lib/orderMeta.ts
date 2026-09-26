export const orderStatusLabel: Record<string, string> = {
  pending: '待接单',
  in_transit: '在途',
  partial: '部分到货',
  completed: '已完成',
  rejected: '无法供货',
  cancelled: '已取消',
};
export const orderStatusTone: Record<string, 'neutral' | 'good' | 'alert'> = {
  pending: 'neutral',
  in_transit: 'good',
  partial: 'good',
  completed: 'good',
  rejected: 'alert',
  cancelled: 'neutral',
};
export const orderTabs: Array<{ value: string; label: string }> = [
  { value: '', label: '全部' },
  { value: 'pending', label: '待接单' },
  { value: 'in_transit', label: '在途' },
  { value: 'partial', label: '部分到货' },
  { value: 'completed', label: '已完成' },
];
