import { AuthProvider } from './auth/AuthProvider';
import AppRouter from './routing/AppRouter';

export default function App() {
  return (
    <AuthProvider>
      <AppRouter />
    </AuthProvider>
  );
}
