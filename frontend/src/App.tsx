import { GoogleOAuthProvider } from '@react-oauth/google';
import { Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import Login from './components/Login';
import Header from './components/Header';
import TournamentList from './components/TournamentList';
import TournamentView from './components/TournamentView';
import './App.css';

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID || '';
const DEV_MODE = import.meta.env.VITE_DEV_MODE === 'true';

function AppContent() {
  const { isAuthenticated } = useAuth();

  if (!isAuthenticated) {
    return <Login devMode={DEV_MODE} />;
  }

  return (
    <>
      <Header />
      <main className="container">
        <Routes>
          <Route path="/" element={<TournamentList />} />
          <Route path="/tournament/:id" element={<Navigate to="scoreboard" replace />} />
          <Route path="/tournament/:id/:tab" element={<TournamentView />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </>
  );
}

export default function App() {
  // In dev mode, skip Google OAuth entirely
  if (DEV_MODE) {
    return (
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    );
  }

  if (!GOOGLE_CLIENT_ID) {
    return (
      <div className="login-container">
        <div className="login-card">
          <h1>Configuration Required</h1>
          <p>
            Set <code>VITE_GOOGLE_CLIENT_ID</code> in a <code>.env</code> file,
            or set <code>VITE_DEV_MODE=true</code> to skip authentication.
          </p>
          <pre>VITE_GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com</pre>
        </div>
      </div>
    );
  }

  return (
    <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </GoogleOAuthProvider>
  );
}
