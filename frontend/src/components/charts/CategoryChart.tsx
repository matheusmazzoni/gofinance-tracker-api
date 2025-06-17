import { useMemo } from 'react';
import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import type { Transaction, Category } from '../../types/api';
import { formatCurrency } from '../../utils/formatters';

interface CategoryChartProps {
  transactions: Transaction[];
  categories: Category[];
  type: 'income' | 'expense';
}

const COLORS = [
  '#1976d2',
  '#2196f3',
  '#42a5f5',
  '#64b5f6',
  '#90caf9',
  '#bbdefb',
  '#e3f2fd',
];

export default function CategoryChart({
  transactions,
  categories,
  type,
}: CategoryChartProps) {
  const data = useMemo(() => {
    const categoryMap = new Map<string, number>();

    transactions
      .filter((t) => (type === 'income' ? t.amount > 0 : t.amount < 0))
      .forEach((transaction) => {
        const categoryId = transaction.categoryId;
        const currentAmount = categoryMap.get(categoryId) || 0;
        categoryMap.set(
          categoryId,
          currentAmount + Math.abs(transaction.amount)
        );
      });

    return Array.from(categoryMap.entries())
      .map(([categoryId, amount]) => {
        const category = categories.find((c) => c.id === categoryId);
        return {
          name: category?.name || 'Sem categoria',
          value: amount,
        };
      })
      .sort((a, b) => b.value - a.value);
  }, [transactions, categories, type]);

  return (
    <ResponsiveContainer width="100%" height={400}>
      <PieChart>
        <Pie
          data={data}
          dataKey="value"
          nameKey="name"
          cx="50%"
          cy="50%"
          outerRadius={150}
          label={({ name, percent }) =>
            `${name} (${(percent * 100).toFixed(1)}%)`
          }
        >
          {data.map((_, index) => (
            <Cell
              key={`cell-${index}`}
              fill={COLORS[index % COLORS.length]}
            />
          ))}
        </Pie>
        <Tooltip
          formatter={(value: number) => formatCurrency(value)}
          labelStyle={{ color: '#666' }}
        />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  );
} 