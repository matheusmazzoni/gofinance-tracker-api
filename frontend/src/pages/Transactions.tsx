import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Button,
  Card,
  CardContent,
  IconButton,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Stack,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import { transactionService, accountService, categoryService } from '../services/api';
import { formatCurrency, formatDate } from '../utils/formatters';
import type { Transaction, Account, Category, ApiResponse } from '../types/api';

export default function Transactions() {
  const [open, setOpen] = useState(false);
  const [selectedTransaction, setSelectedTransaction] = useState<Transaction | null>(null);
  const queryClient = useQueryClient();

  const { data: transactionsResponse, isLoading: isLoadingTransactions } = useQuery<ApiResponse<Transaction[]>>({
    queryKey: ['transactions'],
    queryFn: () => transactionService.getAll(),
  });

  const { data: accountsResponse } = useQuery<ApiResponse<Account[]>>({
    queryKey: ['accounts'],
    queryFn: () => accountService.getAll(),
  });

  const { data: categoriesResponse } = useQuery<ApiResponse<Category[]>>({
    queryKey: ['categories'],
    queryFn: () => categoryService.getAll(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => transactionService.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
    },
  });

  const handleOpen = (transaction?: Transaction) => {
    setSelectedTransaction(transaction || null);
    setOpen(true);
  };

  const handleClose = () => {
    setOpen(false);
    setSelectedTransaction(null);
  };

  const handleDelete = (id: string) => {
    if (window.confirm('Tem certeza que deseja excluir esta transação?')) {
      deleteMutation.mutate(id);
    }
  };

  if (isLoadingTransactions) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  const transactions = transactionsResponse?.data || [];
  const accounts = accountsResponse?.data || [];
  const categories = categoriesResponse?.data || [];

  return (
    <Box p={{ xs: 1, md: 3 }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" mb={3}>
        <Typography variant="h4" fontWeight={700}>Minhas Transações</Typography>
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
                  <TableCell>Data</TableCell>
                  <TableCell>Descrição</TableCell>
                  <TableCell align="right">Valor</TableCell>
                  <TableCell>Tipo</TableCell>
                  <TableCell>Categoria</TableCell>
                  <TableCell align="center">Ações</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {transactions.map((transaction) => (
                  <TableRow key={transaction.id}>
                    <TableCell>{formatDate(transaction.date)}</TableCell>
                    <TableCell>{transaction.description}</TableCell>
                    <TableCell align="right" sx={{ color: transaction.amount >= 0 ? 'success.main' : 'error.main', fontWeight: 600 }}>
                      {transaction.amount >= 0 ? '+' : '-'}{formatCurrency(Math.abs(transaction.amount))}
                    </TableCell>
                    <TableCell>{transaction.amount >= 0 ? 'Receita' : 'Despesa'}</TableCell>
                    <TableCell>{categories.find((c) => c.id === transaction.categoryId)?.name || '-'}</TableCell>
                    <TableCell align="center">
                      <IconButton size="small" color="primary" onClick={() => handleOpen(transaction)}>
                        <EditIcon />
                      </IconButton>
                      <IconButton size="small" color="error" onClick={() => handleDelete(transaction.id)}>
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
        <DialogTitle>{selectedTransaction ? 'Editar Transação' : 'Adicionar Transação'}</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              label="Descrição"
              fullWidth
              defaultValue={selectedTransaction?.description}
            />
            <TextField
              label="Valor"
              type="number"
              fullWidth
              defaultValue={selectedTransaction?.amount}
            />
            <TextField
              label="Data"
              type="date"
              fullWidth
              defaultValue={selectedTransaction?.date?.split('T')[0]}
              InputLabelProps={{ shrink: true }}
            />
            <TextField
              select
              label="Categoria"
              fullWidth
              defaultValue={selectedTransaction?.categoryId}
            >
              {categories.map((category) => (
                <MenuItem key={category.id} value={category.id}>
                  {category.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              select
              label="Conta"
              fullWidth
              defaultValue={selectedTransaction?.accountId}
            >
              {accounts.map((account) => (
                <MenuItem key={account.id} value={account.id}>
                  {account.name}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Notas"
              multiline
              rows={3}
              fullWidth
              defaultValue={selectedTransaction?.notes}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Cancelar</Button>
          <Button variant="contained" onClick={handleClose}>
            {selectedTransaction ? 'Salvar' : 'Adicionar'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
} 