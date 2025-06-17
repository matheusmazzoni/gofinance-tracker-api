import { api } from './api';
import type { Category, ApiResponse } from '../types/api';

export async function getCategories(): Promise<ApiResponse<Category[]>> {
  const response = await api.get<ApiResponse<Category[]>>('/categories');
  return response.data;
}

export async function createCategory(category: Omit<Category, 'id'>): Promise<ApiResponse<Category>> {
  const response = await api.post<ApiResponse<Category>>('/categories', category);
  return response.data;
}

export async function updateCategory(id: string, category: Partial<Category>): Promise<ApiResponse<Category>> {
  const response = await api.put<ApiResponse<Category>>(`/categories/${id}`, category);
  return response.data;
}

export async function deleteCategory(id: string): Promise<ApiResponse<void>> {
  const response = await api.delete<ApiResponse<void>>(`/categories/${id}`);
  return response.data;
} 