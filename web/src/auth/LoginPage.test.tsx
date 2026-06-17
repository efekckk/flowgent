import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi } from 'vitest';
import LoginPage from './LoginPage';
import { AuthContext, type AuthState } from './AuthProvider';

function renderWithAuth(login = vi.fn()) {
  const value: AuthState = {
    user: null, loading: false,
    signup: vi.fn(), login, logout: vi.fn(),
  };
  return render(
    <AuthContext.Provider value={value}>
      <MemoryRouter><LoginPage /></MemoryRouter>
    </AuthContext.Provider>,
  );
}

describe('LoginPage', () => {
  it('renders email and password fields', () => {
    renderWithAuth();
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/passphrase/i)).toBeInTheDocument();
  });

  it('calls login() with submitted values', async () => {
    const login = vi.fn().mockResolvedValue(undefined);
    renderWithAuth(login);
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'a@example.com' } });
    fireEvent.change(screen.getByLabelText(/passphrase/i), { target: { value: 'longerpw' } });
    fireEvent.click(screen.getByRole('button', { name: /open sheet/i }));
    await waitFor(() => expect(login).toHaveBeenCalledWith('a@example.com', 'longerpw'));
  });
});
