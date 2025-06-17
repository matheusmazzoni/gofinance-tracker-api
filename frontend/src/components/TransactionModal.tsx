import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  Button,
  Box,
} from '@mui/material';
import type { Transaction, Account, Category } from '../types/api';

interface TransactionModalProps {
  open: boolean;
  onClose: () => void;
  transaction?: Transaction | null;
  type: 'income' | 'expense' | 'transfer';
  accounts: Account[];
  categories: Category[];
}

export default function TransactionModal({
  open,
  onClose,
  transaction,
  type,
  accounts,
  categories,
}: TransactionModalProps) {
  // Filtrar categorias conforme o tipo
  const filteredCategories = type === 'income'
    ? categories.filter((c) => c.type === 'income')
    : type === 'expense'
      ? categories.filter((c) => c.type === 'expense')
      : [];

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {transaction ? 'Editar ' : 'Nova '}
        {type === 'income' ? 'Receita' : type === 'expense' ? 'Despesa' : 'Transferência'}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ pt: 2, display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label="Descrição"
            fullWidth
            defaultValue={transaction?.description}
          />
          <TextField
            label="Valor"
            type="number"
            fullWidth
            defaultValue={transaction?.amount}
          />
          <TextField
            label="Data"
            type="date"
            fullWidth
            defaultValue={transaction?.date?.split('T')[0]}
            InputLabelProps={{ shrink: true }}
          />
          {/* Categoria só para receita/despesa */}
          {type !== 'transfer' && (
            <TextField
              select
              label="Categoria"
              fullWidth
              defaultValue={transaction?.categoryId}
            >
              {filteredCategories.map((category) => (
                <MenuItem key={category.id} value={category.id}>
                  {category.name}
                </MenuItem>
              ))}
            </TextField>
          )}
          {/* Conta(s) */}
          {type === 'transfer' ? (
            <>
              <TextField
                select
                label="Conta de Origem"
                fullWidth
                defaultValue={transaction?.accountId}
              >
                {accounts.map((account) => (
                  <MenuItem key={account.id} value={account.id}>
                    {account.name}
                  </MenuItem>
                ))}
              </TextField>
              <TextField
                select
                label="Conta de Destino"
                fullWidth
              >
                {accounts.map((account) => (
                  <MenuItem key={account.id} value={account.id}>
                    {account.name}
                  </MenuItem>
                ))}
              </TextField>
            </>
          ) : (
            <TextField
              select
              label="Conta"
              fullWidth
              defaultValue={transaction?.accountId}
            >
              {accounts.map((account) => (
                <MenuItem key={account.id} value={account.id}>
                  {account.name}
                </MenuItem>
              ))}
            </TextField>
          )}
          <TextField
            label="Notas"
            multiline
            rows={3}
            fullWidth
            defaultValue={transaction?.notes}
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancelar</Button>
        <Button variant="contained" onClick={onClose}>
          {transaction ? 'Salvar' : 'Adicionar'}
        </Button>
      </DialogActions>
    </Dialog>
  );
} 