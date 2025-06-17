import { createTheme } from '@mui/material/styles';

const theme = createTheme({
  palette: {
    primary: {
      main: '#1A237E',
      contrastText: '#fff',
    },
    secondary: {
      main: '#1976d2',
    },
    success: {
      main: '#43A047',
      contrastText: '#fff',
    },
    error: {
      main: '#E53935',
      contrastText: '#fff',
    },
    warning: {
      main: '#FFB300',
      contrastText: '#fff',
    },
    background: {
      default: '#f5f6fa',
      paper: '#fff',
    },
    text: {
      primary: '#222',
      secondary: '#555',
    },
  },
  typography: {
    fontFamily: [
      'Inter',
      '-apple-system',
      'BlinkMacSystemFont',
      '"Segoe UI"',
      'Roboto',
      '"Helvetica Neue"',
      'Arial',
      'sans-serif',
    ].join(','),
    fontWeightBold: 700,
    fontWeightMedium: 600,
  },
  components: {
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 16,
          boxShadow: '0 4px 16px rgba(0,0,0,0.08)',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          textTransform: 'none',
          fontWeight: 600,
        },
      },
    },
  },
});

export default theme; 