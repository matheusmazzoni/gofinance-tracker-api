import axios from 'axios';
import type { Transaction, Account, Category, ApiResponse } from '../types/api';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const transactionService = {
  getAll: async (): Promise<ApiResponse<Transaction[]>> => {
    const response = await api.get('/transactions');
    return response.data;
  },

  create: async (transaction: Omit<Transaction, 'id'>): Promise<ApiResponse<Transaction>> => {
    const response = await api.post('/transactions', transaction);
    return response.data;
  },

  update: async (id: string, transaction: Partial<Transaction>): Promise<ApiResponse<Transaction>> => {
    const response = await api.put(`/transactions/${id}`, transaction);
    return response.data;
  },

  delete: async (id: string): Promise<ApiResponse<void>> => {
    const response = await api.delete(`/transactions/${id}`);
    return response.data;
  },
};

export const accountService = {
  getAll: async (): Promise<ApiResponse<Account[]>> => {
    const response = await api.get('/accounts');
    return response.data;
  },

  create: async (account: Omit<Account, 'id'>): Promise<ApiResponse<Account>> => {
    const response = await api.post('/accounts', account);
    return response.data;
  },

  update: async (id: string, account: Partial<Account>): Promise<ApiResponse<Account>> => {
    const response = await api.put(`/accounts/${id}`, account);
    return response.data;
  },

  delete: async (id: string): Promise<ApiResponse<void>> => {
    const response = await api.delete(`/accounts/${id}`);
    return response.data;
  },
};

export const categoryService = {
  getAll: async (): Promise<ApiResponse<Category[]>> => {
    const response = await api.get('/categories');
    return response.data;
  },

  create: async (category: Omit<Category, 'id'>): Promise<ApiResponse<Category>> => {
    const response = await api.post('/categories', category);
    return response.data;
  },

  update: async (id: string, category: Partial<Category>): Promise<ApiResponse<Category>> => {
    const response = await api.put(`/categories/${id}`, category);
    return response.data;
  },

  delete: async (id: string): Promise<ApiResponse<void>> => {
    const response = await api.delete(`/categories/${id}`);
    return response.data;
  },
}; 