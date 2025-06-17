import { useMemo } from 'react';
import {
  BarChart,
  Bar,
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

interface IncomeExpenseChartProps {
  transactions: Transaction[];
  months?: number;
}

export default function IncomeExpenseChart({
  transactions,
  months = 6,
}: IncomeExpenseChartProps) {
  const data = useMemo(() => {
    const result = [];
    const now = new Date();

    for (let i = months - 1; i >= 0; i--) {
      const date = subMonths(now, i);
      const monthStart = startOfMonth(date);
      const monthEnd = endOfMonth(date);

      const monthTransactions = transactions.filter(
        (t) => new Date(t.date) >= monthStart && new Date(t.date) <= monthEnd
      );

      const income = monthTransactions
        .filter((t) => t.amount > 0)
        .reduce((sum, t) => sum + t.amount, 0);

      const expenses = monthTransactions
        .filter((t) => t.amount < 0)
        .reduce((sum, t) => sum + Math.abs(t.amount), 0);

      result.push({
        month: format(date, 'MMM yyyy', { locale: ptBR }),
        income,
        expenses,
        balance: income - expenses,
      });
    }

    return result;
  }, [transactions, months]);

  return (
    <ResponsiveContainer width="100%" height={400}>
      <BarChart data={data}>
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
        <Bar
          dataKey="income"
          name="Receitas"
          fill="#2e7d32"
          radius={[4, 4, 0, 0]}
        />
        <Bar
          dataKey="expenses"
          name="Despesas"
          fill="#c62828"
          radius={[4, 4, 0, 0]}
        />
      </BarChart>
    </ResponsiveContainer>
  );
} 