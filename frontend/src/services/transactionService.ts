import { api } from './api';
import type { Transaction, ApiResponse } from '../types/api';

export async function getTransactions(): Promise<ApiResponse<Transaction[]>> {
  const response = await api.get<ApiResponse<Transaction[]>>('/transactions');
  return response.data;
}

export async function createTransaction(transaction: Omit<Transaction, 'id'>): Promise<ApiResponse<Transaction>> {
  const response = await api.post<ApiResponse<Transaction>>('/transactions', transaction);
  return response.data;
}

export async function updateTransaction(id: string, transaction: Partial<Transaction>): Promise<ApiResponse<Transaction>> {
  const response = await api.put<ApiResponse<Transaction>>(`/transactions/${id}`, transaction);
  return response.data;
}

export async function deleteTransaction(id: string): Promise<ApiResponse<void>> {
  const response = await api.delete<ApiResponse<void>>(`/transactions/${id}`);
  return response.data;
} 