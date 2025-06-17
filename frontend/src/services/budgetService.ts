import { api } from './api';
import type { Budget, ApiResponse } from '../types/api';

export async function getBudgets(): Promise<ApiResponse<Budget[]>> {
  const { data } = await api.get('/budgets');
  return data;
}

export async function createBudget(budget: Omit<Budget, 'id' | 'createdAt' | 'updatedAt' | 'categoryName' | 'spent' | 'remaining' | 'status'>): Promise<ApiResponse<Budget>> {
  const { data } = await api.post('/budgets', budget);
  return data;
}

export async function updateBudget(id: string, budget: Partial<Budget>): Promise<ApiResponse<Budget>> {
  const { data } = await api.put(`/budgets/${id}`, budget);
  return data;
}

export async function deleteBudget(id: string): Promise<ApiResponse<Budget>> {
  const { data } = await api.delete(`/budgets/${id}`);
  return data;
} 