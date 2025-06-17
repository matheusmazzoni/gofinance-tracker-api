import { api } from './api';
import type { Account, ApiResponse } from '../types/api';

export async function getAccounts(): Promise<ApiResponse<Account[]>> {
  const response = await api.get<ApiResponse<Account[]>>('/accounts');
  return response.data;
}

export async function createAccount(account: Omit<Account, 'id'>): Promise<ApiResponse<Account>> {
  const response = await api.post<ApiResponse<Account>>('/accounts', account);
  return response.data;
}

export async function updateAccount(id: string, account: Partial<Account>): Promise<ApiResponse<Account>> {
  const response = await api.put<ApiResponse<Account>>(`/accounts/${id}`, account);
  return response.data;
}

export async function deleteAccount(id: string): Promise<ApiResponse<void>> {
  const response = await api.delete<ApiResponse<void>>(`/accounts/${id}`);
  return response.data;
} 