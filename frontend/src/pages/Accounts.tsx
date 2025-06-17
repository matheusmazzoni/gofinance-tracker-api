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
  Switch,
  TextField,
  Typography,
  CircularProgress,
  Chip,
  Stack,
} from '@mui/material';
import { Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { DataGrid } from '@mui/x-data-grid';
import type { GridRenderCellParams, GridColDef } from '@mui/x-data-grid';
import { getAccounts, createAccount, updateAccount, deleteAccount } from '../services/accountService';
import { formatCurrency } from '../utils/formatters';
import type { Account, ApiResponse } from '../types/api';

const accountTypes = [
  { value: 'checking', label: 'Conta Corrente' },
  { value: 'savings', label: 'Conta Poupança' },
  { value: 'investment', label: 'Investimento' },
  { value: 'credit', label: 'Cartão de Crédito' },
  { value: 'other', label: 'Outro' },
];

export default function Accounts() {
  const [open, setOpen] = useState(false);
  const [selectedAccount, setSelectedAccount] = useState<Account | null>(null);
  const queryClient = useQueryClient();

  const { data: accountsResponse, isLoading } = useQuery<ApiResponse<Account[]>>({
    queryKey: ['accounts'],
    queryFn: getAccounts,
  });

  const createMutation = useMutation({
    mutationFn: createAccount,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      handleClose();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Account> }) =>
      updateAccount(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      handleClose();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteAccount,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
  });

  const handleOpen = (account?: Account) => {
    setSelectedAccount(account || null);
    setOpen(true);
  };

  const handleClose = () => {
    setSelectedAccount(null);
    setOpen(false);
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const data = {
      name: formData.get('name') as string,
      type: formData.get('type') as Account['type'],
      balance: Number(formData.get('balance')),
      description: formData.get('description') as string,
      isActive: formData.get('isActive') === 'true',
    };
    if (selectedAccount) {
      updateMutation.mutate({ id: selectedAccount.id, data: data as any });
    } else {
      createMutation.mutate(data as any);
    }
  };

  const handleDelete = (id: string) => {
    if (window.confirm('Tem certeza que deseja excluir esta conta?')) {
      deleteMutation.mutate(id);
    }
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  const columns: GridColDef[] = [
    { field: 'name', headerName: 'Nome', flex: 1 },
    {
      field: 'type',
      headerName: 'Tipo',
      flex: 1,
      valueGetter: (params: any) =>
        accountTypes.find((t) => t.value === params.value)?.label || params.value,
    },
    {
      field: 'balance',
      headerName: 'Saldo',
      flex: 1,
      valueFormatter: (params) => formatCurrency(params.value as number),
      renderCell: (params: GridRenderCellParams) => (
        <Typography fontWeight={600} color={params.value > 0 ? 'success.main' : 'error.main'}>
          {formatCurrency(params.value as number)}
        </Typography>
      ),
    },
    {
      field: 'isActive',
      headerName: 'Status',
      flex: 1,
      renderCell: (params: any) => (
        <Chip
          label={params.value ? 'Ativo' : 'Inativo'}
          size="small"
          sx={{
            bgcolor: params.value ? 'success.main' : 'grey.400',
            color: '#fff',
            fontWeight: 600,
          }}
        />
      ),
    },
    {
      field: 'actions',
      headerName: 'Ações',
      flex: 1,
      sortable: false,
      renderCell: (params) => (
        <Box>
          <IconButton
            color="primary"
            onClick={() => handleOpen(params.row)}
            size="small"
          >
            <EditIcon />
          </IconButton>
          <IconButton
            color="error"
            onClick={() => handleDelete(params.row.id)}
            size="small"
          >
            <DeleteIcon />
          </IconButton>
        </Box>
      ),
    },
  ];

  return (
    <Box p={{ xs: 1, md: 3 }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" mb={3}>
        <Typography variant="h4" fontWeight={700}>Minhas Contas</Typography>
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
          <DataGrid
            rows={accountsResponse?.data || []}
            columns={columns}
            autoHeight
            pageSizeOptions={[10, 25, 50]}
            initialState={{
              pagination: { paginationModel: { pageSize: 10 } },
            }}
            disableRowSelectionOnClick
            sx={{
              borderRadius: 2,
              background: '#fff',
              '& .MuiDataGrid-columnHeaders': { fontWeight: 700 },
              '& .MuiDataGrid-cell': { fontSize: 15 },
              '& .MuiDataGrid-row': { minHeight: 48 },
            }}
          />
        </CardContent>
      </Card>

      <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
        <form onSubmit={handleSubmit}>
          <DialogTitle>
            {selectedAccount ? 'Editar Conta' : 'Nova Conta'}
          </DialogTitle>
          <DialogContent>
            <Box display="flex" flexDirection="column" gap={2} mt={1}>
              <TextField
                name="name"
                label="Nome"
                fullWidth
                required
                defaultValue={selectedAccount?.name}
              />
              <TextField
                name="type"
                label="Tipo"
                select
                fullWidth
                required
                defaultValue={selectedAccount?.type}
              >
                {accountTypes.map((type) => (
                  <MenuItem key={type.value} value={type.value}>
                    {type.label}
                  </MenuItem>
                ))}
              </TextField>
              <TextField
                name="balance"
                label="Saldo"
                type="number"
                fullWidth
                required
                defaultValue={selectedAccount?.balance}
              />
              <TextField
                name="description"
                label="Descrição"
                multiline
                rows={4}
                fullWidth
                defaultValue={selectedAccount?.description}
              />
              <Box display="flex" alignItems="center">
                <Typography>Ativo</Typography>
                <Switch
                  name="isActive"
                  defaultChecked={selectedAccount?.isActive ?? true}
                />
              </Box>
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={handleClose}>Cancelar</Button>
            <Button type="submit" variant="contained">
              {selectedAccount ? 'Salvar' : 'Criar'}
            </Button>
          </DialogActions>
        </form>
      </Dialog>
    </Box>
  );
} 