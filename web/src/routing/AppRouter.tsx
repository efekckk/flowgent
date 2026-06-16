import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import LoginPage from '../auth/LoginPage';
import SignupPage from '../auth/SignupPage';
import WorkflowsPage from '../workflows/WorkflowsPage';
import ProtectedRoute from './ProtectedRoute';
import CredentialsPage from '../credentials/CredentialsPage';
import TriggersPage from '../triggers/TriggersPage';
import RunListPage from '../runs/RunListPage';

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/signup" element={<SignupPage />} />
        <Route
          path="/workflows"
          element={<ProtectedRoute><WorkflowsPage /></ProtectedRoute>}
        />
        <Route
          path="/workflows/:id"
          element={<ProtectedRoute><WorkflowsPage /></ProtectedRoute>}
        />
        <Route
          path="/credentials"
          element={<ProtectedRoute><CredentialsPage /></ProtectedRoute>}
        />
        <Route
          path="/workflows/:id/triggers"
          element={<ProtectedRoute><TriggersPage /></ProtectedRoute>}
        />
        <Route
          path="/workflows/:id/runs"
          element={<ProtectedRoute><RunListPage /></ProtectedRoute>}
        />
        <Route path="*" element={<Navigate to="/workflows" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
