import { useQuery } from '@tanstack/react-query';
import {
  Box,
  Card,
  CardContent,
  Typography,
  CircularProgress,
  Grid,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Avatar,
  Stack,
} from '@mui/material';
import { Add, Remove, AccountBalanceWallet } from '@mui/icons-material';
import { getTransactions } from '../services/transactionService';
import { getAccounts } from '../services/accountService';
import { getCategories } from '../services/categoryService';
import { formatCurrency } from '../utils/formatters';
import CategoryChart from '../components/charts/CategoryChart';
import type { Transaction, Account, Category, ApiResponse } from '../types/api';

export default function Dashboard() {
  const { data: transactionsResponse, isLoading: isLoadingTransactions } = useQuery<ApiResponse<Transaction[]>>({
    queryKey: ['transactions'],
    queryFn: getTransactions,
  });

  const { data: accountsResponse, isLoading: isLoadingAccounts } = useQuery<ApiResponse<Account[]>>({
    queryKey: ['accounts'],
    queryFn: getAccounts,
  });

  const { data: categoriesResponse, isLoading: isLoadingCategories } = useQuery<ApiResponse<Category[]>>({
    queryKey: ['categories'],
    queryFn: getCategories,
  });

  const isLoading = isLoadingTransactions || isLoadingAccounts || isLoadingCategories;

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  const transactions = transactionsResponse?.data || [];
  const accounts = accountsResponse?.data || [];
  const categories = categoriesResponse?.data || [];

  const totalBalance = accounts.reduce((sum: number, account: Account) => sum + account.balance, 0);
  const monthlyIncome = transactions.filter((t: Transaction) => t.amount > 0).reduce((sum: number, t: Transaction) => sum + t.amount, 0);
  const monthlyExpenses = transactions.filter((t: Transaction) => t.amount < 0).reduce((sum: number, t: Transaction) => sum + Math.abs(t.amount), 0);

  const latestTransactions = transactions.slice(0, 5);

  return (
    <Box p={{ xs: 1, md: 3 }}>
      <Typography variant="h4" fontWeight={700} mb={3}>
        Painel Principal
      </Typography>
      <Grid container spacing={3} mb={2}>
        {/* Card Saldo Total */}
        <Grid item xs={12} md={4}>
          <Card sx={{ borderRadius: 3, boxShadow: 2 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between">
                <Box>
                  <Typography color="text.secondary" fontWeight={600} gutterBottom>
                    Saldo Total
                  </Typography>
                  <Typography variant="h4" fontWeight={700}>
                    {formatCurrency(totalBalance)}
                  </Typography>
                </Box>
                <Avatar sx={{ bgcolor: 'primary.main', width: 48, height: 48 }}>
                  <AccountBalanceWallet />
                </Avatar>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
        {/* Card Receitas */}
        <Grid item xs={12} md={4}>
          <Card sx={{ borderRadius: 3, boxShadow: 2 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between">
                <Box>
                  <Typography color="text.secondary" fontWeight={600} gutterBottom>
                    Total de Receitas
                  </Typography>
                  <Typography variant="h4" fontWeight={700} color="success.main">
                    {formatCurrency(monthlyIncome)}
                  </Typography>
                </Box>
                <Avatar sx={{ bgcolor: 'success.main', width: 48, height: 48 }}>
                  <Add />
                </Avatar>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
        {/* Card Despesas */}
        <Grid item xs={12} md={4}>
          <Card sx={{ borderRadius: 3, boxShadow: 2 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between">
                <Box>
                  <Typography color="text.secondary" fontWeight={600} gutterBottom>
                    Total de Despesas
                  </Typography>
                  <Typography variant="h4" fontWeight={700} color="error.main">
                    {formatCurrency(monthlyExpenses)}
                  </Typography>
                </Box>
                <Avatar sx={{ bgcolor: 'error.main', width: 48, height: 48 }}>
                  <Remove />
                </Avatar>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Gráfico de Despesas por Categoria */}
      <Grid container spacing={3} mb={2}>
        <Grid item xs={12} md={8}>
          <Card sx={{ borderRadius: 3, boxShadow: 2, height: '100%' }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between" mb={2}>
                <Typography variant="h6" fontWeight={700}>
                  Despesas por Categoria
                </Typography>
                <Button variant="contained" color="primary" size="small">
                  Gerir Categorias
                </Button>
              </Stack>
              <CategoryChart transactions={transactions} categories={categories} type="expense" />
            </CardContent>
          </Card>
        </Grid>
        {/* Últimas Transações */}
        <Grid item xs={12} md={4}>
          <Card sx={{ borderRadius: 3, boxShadow: 2, height: '100%' }}>
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between" mb={2}>
                <Typography variant="h6" fontWeight={700}>
                  Últimas Transações
                </Typography>
                <Button variant="outlined" color="primary" size="small">
                  Ver Todas as Transações
                </Button>
              </Stack>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Data</TableCell>
                      <TableCell>Descrição</TableCell>
                      <TableCell align="right">Valor</TableCell>
                      <TableCell>Categoria</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {latestTransactions.map((t) => (
                      <TableRow key={t.id}>
                        <TableCell>{t.date?.slice(0, 10)}</TableCell>
                        <TableCell>{t.description}</TableCell>
                        <TableCell align="right" sx={{ color: t.amount > 0 ? 'success.main' : 'error.main', fontWeight: 600 }}>
                          {t.amount > 0 ? '+' : '-'}{formatCurrency(Math.abs(t.amount))}
                        </TableCell>
                        <TableCell>{categories.find((c) => c.id === t.categoryId)?.name || '-'}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
} 