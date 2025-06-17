import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Button,
  Card,
  CardContent,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  MenuItem,
  TextField,
  Typography,
  CircularProgress,
  Chip,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import { Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { getBudgets, createBudget, updateBudget, deleteBudget } from '../services/budgetService';
import { getCategories } from '../services/categoryService';
import { formatCurrency } from '../utils/formatters';
import type { Budget, Category, ApiResponse } from '../types/api';

export default function Budgets() {
  const [open, setOpen] = useState(false);
  const [selectedBudget, setSelectedBudget] = useState<Budget | null>(null);
  const queryClient = useQueryClient();

  const { data: budgetsResponse, isLoading: isLoadingBudgets } = useQuery<ApiResponse<Budget[]>>({
    queryKey: ['budgets'],
    queryFn: getBudgets,
  });

  const { data: categoriesResponse, isLoading: isLoadingCategories } = useQuery<ApiResponse<Category[]>>({
    queryKey: ['categories'],
    queryFn: getCategories,
  });

  const createMutation = useMutation({
    mutationFn: createBudget,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgets'] });
      handleClose();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Budget> }) => updateBudget(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgets'] });
      handleClose();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteBudget,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgets'] });
    },
  });

  const handleOpen = (budget?: Budget) => {
    setSelectedBudget(budget || null);
    setOpen(true);
  };

  const handleClose = () => {
    setSelectedBudget(null);
    setOpen(false);
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const data = {
      categoryId: formData.get('categoryId') as string,
      allocated: Number(formData.get('allocated')),
    };
    if (selectedBudget) {
      updateMutation.mutate({ id: selectedBudget.id, data: data as any });
    } else {
      createMutation.mutate(data as any);
    }
  };

  const handleDelete = (id: string) => {
    if (window.confirm('Tem certeza que deseja excluir este orçamento?')) {
      deleteMutation.mutate(id);
    }
  };

  if (isLoadingBudgets || isLoadingCategories) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  const budgets = budgetsResponse?.data || [];
  const categories = categoriesResponse?.data || [];

  const statusColor = (status: string) => {
    if (status === 'OK') return 'success.main';
    if (status === 'Estourado') return 'error.main';
    if (status === 'Atenção') return 'warning.main';
    return 'grey.400';
  };

  return (
    <Box p={{ xs: 1, md: 3 }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" mb={3}>
        <Typography variant="h4" fontWeight={700}>Meus Orçamentos</Typography>
        <Button
          variant="contained"
          color="success"
          startIcon={<AddIcon />}
          onClick={() => handleOpen()}
        >
          Adicionar
        </Button>
      </Stack>

      <Card sx={{ borderRadius: 3, boxShadow: 2 }}>
        <CardContent>
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Nome da Categoria</TableCell>
                  <TableCell>Alocado</TableCell>
                  <TableCell>Gasto</TableCell>
                  <TableCell>Restante</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell align="center">Ações</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {budgets.map((budget) => (
                  <TableRow key={budget.id}>
                    <TableCell>{budget.categoryName}</TableCell>
                    <TableCell>{formatCurrency(budget.allocated)}</TableCell>
                    <TableCell>{formatCurrency(budget.spent)}</TableCell>
                    <TableCell sx={{ color: budget.remaining < 0 ? 'error.main' : 'success.main', fontWeight: 600 }}>
                      {formatCurrency(budget.remaining)}
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={budget.status}
                        size="small"
                        sx={{
                          bgcolor: statusColor(budget.status),
                          color: '#fff',
                          fontWeight: 600,
                        }}
                      />
                    </TableCell>
                    <TableCell align="center">
                      <IconButton size="small" color="primary" onClick={() => handleOpen(budget)}>
                        <EditIcon />
                      </IconButton>
                      <IconButton size="small" color="error" onClick={() => handleDelete(budget.id)}>
                        <DeleteIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
        <form onSubmit={handleSubmit}>
          <DialogTitle>{selectedBudget ? 'Editar Orçamento' : 'Novo Orçamento'}</DialogTitle>
          <DialogContent>
            <Box display="flex" flexDirection="column" gap={2} mt={1}>
              <TextField
                name="categoryId"
                label="Categoria"
                select
                fullWidth
                required
                defaultValue={selectedBudget?.categoryId}
              >
                {categories.map((category) => (
                  <MenuItem key={category.id} value={category.id}>
                    {category.name}
                  </MenuItem>
                ))}
              </TextField>
              <TextField
                name="allocated"
                label="Valor Alocado"
                type="number"
                fullWidth
                required
                defaultValue={selectedBudget?.allocated}
              />
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={handleClose}>Cancelar</Button>
            <Button type="submit" variant="contained">
              {selectedBudget ? 'Salvar' : 'Criar'}
            </Button>
          </DialogActions>
        </form>
      </Dialog>
    </Box>
  );
} 