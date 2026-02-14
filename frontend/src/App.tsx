import { Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import Login from './components/Login';
import Register from './components/Register';
import VerifyEmail from './components/VerifyEmail';
import Header from './components/Header';
import TournamentList from './components/TournamentList';
import TournamentView from './components/TournamentView';
import AdminUsers from './components/AdminUsers';
import './App.css';

const DEV_MODE = import.meta.env.VITE_DEV_MODE === 'true';

function AppContent() {
  const { isAuthenticated, user } = useAuth();

  if (!isAuthenticated) {
    return (
      <Routes>
        <Route path="/register" element={<Register />} />
        <Route path="/verify" element={<VerifyEmail />} />
        <Route path="*" element={<Login devMode={DEV_MODE} />} />
      </Routes>
    );
  }

  return (
    <>
      <Header />
      <main className="container">
        <Routes>
          <Route path="/" element={<TournamentList />} />
          <Route path="/tournament/:id" element={<Navigate to="scoreboard" replace />} />
          <Route path="/tournament/:id/:tab" element={<TournamentView />} />
          {user?.isAdmin && <Route path="/admin/users" element={<AdminUsers />} />}
          <Route path="/verify" element={<VerifyEmail />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}
