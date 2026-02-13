import { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { User } from '../types';
import * as api from '../api/client';

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (accessToken: string) => Promise<void>;
  devLogin: () => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    const stored = localStorage.getItem('user');
    return stored ? JSON.parse(stored) : null;
  });
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('access_token'));

  const login = useCallback(async (accessToken: string) => {
    // Store token first so apiFetch can use it
    localStorage.setItem('access_token', accessToken);
    setToken(accessToken);

    // Fetch user info from our backend (includes isAdmin)
    const me = await api.getMe();

    const userData: User = {
      email: me.email,
      name: me.name,
      picture: me.picture,
      isAdmin: me.isAdmin,
    };

    localStorage.setItem('user', JSON.stringify(userData));
    setUser(userData);
  }, []);

  const devLogin = useCallback(async () => {
    localStorage.setItem('access_token', 'dev-token');
    setToken('dev-token');

    const me = await api.getMe();

    const userData: User = {
      email: me.email,
      name: me.name,
      picture: me.picture,
      isAdmin: me.isAdmin,
    };

    localStorage.setItem('user', JSON.stringify(userData));
    setUser(userData);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('access_token');
    localStorage.removeItem('user');
    setToken(null);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, token, login, devLogin, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
