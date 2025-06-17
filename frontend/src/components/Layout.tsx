import { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  AppBar,
  Box,
  CssBaseline,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
  SpeedDial,
  SpeedDialAction,
  SpeedDialIcon,
} from '@mui/material';
import {
  Menu as MenuIcon,
  Home as HomeIcon,
  AccountBalanceWallet,
  Category,
  PieChart,
  Savings,
  Add,
  Remove,
  SwapHoriz,
} from '@mui/icons-material';
import TransactionModal from './TransactionModal';
import { useQuery } from '@tanstack/react-query';
import { getAccounts } from '../services/accountService';
import { getCategories } from '../services/categoryService';

const drawerWidth = 240;

interface LayoutProps {
  children: React.ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const [modalOpen, setModalOpen] = useState(false);
  const [modalType, setModalType] = useState<'income' | 'expense' | 'transfer'>('income');

  const { data: accountsResponse } = useQuery({ queryKey: ['accounts'], queryFn: getAccounts });
  const { data: categoriesResponse } = useQuery({ queryKey: ['categories'], queryFn: getCategories });

  const accounts = accountsResponse?.data || [];
  const categories = categoriesResponse?.data || [];

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  const menuItems = [
    { text: 'Painel Principal', icon: <HomeIcon />, path: '/' },
    { text: 'Transações', icon: <AccountBalanceWallet />, path: '/transactions' },
    { text: 'Categorias', icon: <Category />, path: '/categories' },
    { text: 'Contas', icon: <PieChart />, path: '/accounts' },
    { text: 'Orçamentos', icon: <Savings />, path: '/budgets' },
  ];

  const drawer = (
    <div style={{ background: '#111827', height: '100%', color: '#fff' }}>
      <Toolbar>
        <Typography variant="h6" noWrap component="div" sx={{ fontWeight: 700, color: '#fff' }}>
          Finanças Pessoais
        </Typography>
      </Toolbar>
      <Divider sx={{ borderColor: 'rgba(255,255,255,0.12)' }} />
      <List>
        {menuItems.map((item) => (
          <ListItem key={item.text} disablePadding>
            <ListItemButton
              selected={location.pathname === item.path}
              onClick={() => {
                navigate(item.path);
                setMobileOpen(false);
              }}
              sx={{
                background: location.pathname === item.path ? '#1A237E' : 'transparent',
                color: '#fff',
                '&:hover': { background: '#1A237E' },
                borderRadius: 2,
                mx: 1,
                my: 0.5,
              }}
            >
              <ListItemIcon sx={{ color: '#fff' }}>{item.icon}</ListItemIcon>
              <ListItemText primary={item.text} />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </div>
  );

  const handleOpenModal = (type: 'income' | 'expense' | 'transfer') => {
    setModalType(type);
    setModalOpen(true);
  };
  const handleCloseModal = () => setModalOpen(false);

  return (
    <Box sx={{ display: 'flex' }}>
      <CssBaseline />
      <AppBar
        position="fixed"
        elevation={0}
        sx={{
          width: { sm: `calc(100% - ${drawerWidth}px)` },
          ml: { sm: `${drawerWidth}px` },
          background: '#fff',
          color: '#1A237E',
          boxShadow: '0 2px 8px rgba(0,0,0,0.04)',
          borderBottom: '1px solid #f0f0f0',
        }}
      >
        <Toolbar>
          <IconButton
            color="inherit"
            aria-label="open drawer"
            edge="start"
            onClick={handleDrawerToggle}
            sx={{ mr: 2, display: { sm: 'none' } }}
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" noWrap component="div" sx={{ fontWeight: 700 }}>
            {menuItems.find((item) => item.path === location.pathname)?.text || 'Finanças Pessoais'}
          </Typography>
        </Toolbar>
      </AppBar>
      <Box
        component="nav"
        sx={{ width: { sm: drawerWidth }, flexShrink: { sm: 0 } }}
      >
        <Drawer
          variant="temporary"
          open={mobileOpen}
          onClose={handleDrawerToggle}
          ModalProps={{
            keepMounted: true, // Better open performance on mobile.
          }}
          sx={{
            display: { xs: 'block', sm: 'none' },
            '& .MuiDrawer-paper': {
              boxSizing: 'border-box',
              width: drawerWidth,
            },
          }}
        >
          {drawer}
        </Drawer>
        <Drawer
          variant="permanent"
          sx={{
            display: { xs: 'none', sm: 'block' },
            '& .MuiDrawer-paper': {
              boxSizing: 'border-box',
              width: drawerWidth,
            },
          }}
          open
        >
          {drawer}
        </Drawer>
      </Box>
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          width: { sm: `calc(100% - ${drawerWidth}px)` },
        }}
      >
        <Toolbar />
        {children}
        <SpeedDial
          ariaLabel="Ações rápidas"
          sx={{ position: 'fixed', bottom: 32, right: 32, zIndex: 1200 }}
          icon={<SpeedDialIcon />}
        >
          <SpeedDialAction
            icon={<Add color="success" />}
            tooltipTitle="Nova Receita"
            onClick={() => handleOpenModal('income')}
          />
          <SpeedDialAction
            icon={<Remove color="error" />}
            tooltipTitle="Nova Despesa"
            onClick={() => handleOpenModal('expense')}
          />
          <SpeedDialAction
            icon={<SwapHoriz color="primary" />}
            tooltipTitle="Nova Transferência"
            onClick={() => handleOpenModal('transfer')}
          />
        </SpeedDial>
        <TransactionModal
          open={modalOpen}
          onClose={handleCloseModal}
          type={modalType}
          accounts={accounts}
          categories={categories}
        />
      </Box>
    </Box>
  );
} 