import { useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { format, subMonths, startOfMonth, endOfMonth } from 'date-fns';
import { ptBR } from 'date-fns/locale';
import type { Transaction } from '../../types/api';
import { formatCurrency } from '../../utils/formatters';

interface BalanceChartProps {
  transactions: Transaction[];
  months?: number;
}

export default function BalanceChart({
  transactions,
  months = 6,
}: BalanceChartProps) {
  const data = useMemo(() => {
    const result = [];
    const now = new Date();
    let runningBalance = 0;

    for (let i = months - 1; i >= 0; i--) {
      const date = subMonths(now, i);
      const monthStart = startOfMonth(date);
      const monthEnd = endOfMonth(date);

      const monthTransactions = transactions.filter(
        (t) => new Date(t.date) >= monthStart && new Date(t.date) <= monthEnd
      );

      const monthBalance = monthTransactions.reduce(
        (sum, t) => sum + t.amount,
        0
      );

      runningBalance += monthBalance;

      result.push({
        month: format(date, 'MMM yyyy', { locale: ptBR }),
        balance: runningBalance,
      });
    }

    return result;
  }, [transactions, months]);

  return (
    <ResponsiveContainer width="100%" height={400}>
      <LineChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="month" />
        <YAxis
          tickFormatter={(value) => formatCurrency(value)}
          width={100}
        />
        <Tooltip
          formatter={(value: number) => formatCurrency(value)}
          labelStyle={{ color: '#666' }}
        />
        <Legend />
        <Line
          type="monotone"
          dataKey="balance"
          name="Saldo"
          stroke="#1976d2"
          strokeWidth={2}
          dot={{ r: 4 }}
          activeDot={{ r: 6 }}
        />
      </LineChart>
    </ResponsiveContainer>
  );
} 