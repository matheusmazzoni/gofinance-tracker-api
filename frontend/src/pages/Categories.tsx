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
import type { GridColDef } from '@mui/x-data-grid';
import { getCategories, createCategory, updateCategory, deleteCategory } from '../services/categoryService';
import type { Category, ApiResponse } from '../types/api';

const categoryTypes = [
  { value: 'income', label: 'Receita' },
  { value: 'expense', label: 'Despesa' },
];

export default function Categories() {
  const [open, setOpen] = useState(false);
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const queryClient = useQueryClient();

  const { data: categoriesResponse, isLoading } = useQuery<ApiResponse<Category[]>>({
    queryKey: ['categories'],
    queryFn: getCategories,
  });

  const createMutation = useMutation({
    mutationFn: createCategory,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      handleClose();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Category> }) =>
      updateCategory(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      handleClose();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteCategory,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
    },
  });

  const handleOpen = (category?: Category) => {
    setSelectedCategory(category || null);
    setOpen(true);
  };

  const handleClose = () => {
    setSelectedCategory(null);
    setOpen(false);
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    const data = {
      name: formData.get('name') as string,
      type: formData.get('type') as Category['type'],
      icon: formData.get('icon') as string,
      color: formData.get('color') as string,
      description: formData.get('description') as string,
      isActive: formData.get('isActive') === 'true',
    };
    if (selectedCategory) {
      updateMutation.mutate({ id: selectedCategory.id, data: data as any });
    } else {
      createMutation.mutate(data as any);
    }
  };

  const handleDelete = (id: string) => {
    if (window.confirm('Tem certeza que deseja excluir esta categoria?')) {
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
        categoryTypes.find((t) => t.value === params.value)?.label || params.value,
    },
    {
      field: 'icon',
      headerName: 'Ícone',
      flex: 1,
    },
    {
      field: 'color',
      headerName: 'Cor',
      flex: 1,
      renderCell: (params) => (
        <Box
          sx={{
            width: 24,
            height: 24,
            borderRadius: '50%',
            backgroundColor: params.value as string,
            border: '1px solid #eee',
          }}
        />
      ),
    },
    {
      field: 'isActive',
      headerName: 'Status',
      flex: 1,
      renderCell: (params) => (
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
        <Typography variant="h4" fontWeight={700}>Minhas Categorias</Typography>
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
            rows={categoriesResponse?.data || []}
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
            {selectedCategory ? 'Editar Categoria' : 'Nova Categoria'}
          </DialogTitle>
          <DialogContent>
            <Box display="flex" flexDirection="column" gap={2} mt={1}>
              <TextField
                name="name"
                label="Nome"
                fullWidth
                required
                defaultValue={selectedCategory?.name}
              />
              <TextField
                name="type"
                label="Tipo"
                select
                fullWidth
                required
                defaultValue={selectedCategory?.type}
              >
                {categoryTypes.map((type) => (
                  <MenuItem key={type.value} value={type.value}>
                    {type.label}
                  </MenuItem>
                ))}
              </TextField>
              <TextField
                name="icon"
                label="Ícone"
                fullWidth
                defaultValue={selectedCategory?.icon}
              />
              <TextField
                name="color"
                label="Cor"
                type="color"
                fullWidth
                defaultValue={selectedCategory?.color || '#000000'}
                InputLabelProps={{ shrink: true }}
              />
              <TextField
                name="description"
                label="Descrição"
                multiline
                rows={4}
                fullWidth
                defaultValue={selectedCategory?.description}
              />
              <Box display="flex" alignItems="center">
                <Typography>Ativo</Typography>
                <Switch
                  name="isActive"
                  defaultChecked={selectedCategory?.isActive ?? true}
                />
              </Box>
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={handleClose}>Cancelar</Button>
            <Button type="submit" variant="contained">
              {selectedCategory ? 'Salvar' : 'Criar'}
            </Button>
          </DialogActions>
        </form>
      </Dialog>
    </Box>
  );
} 