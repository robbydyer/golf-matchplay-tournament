import { lazy, Suspense, useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import * as api from '../api/client';

const GoogleLoginButton = lazy(() => import('./GoogleLoginButton'));

function DevLogin() {
  const { devLogin } = useAuth();

  return (
    <button className="btn btn-primary" onClick={devLogin}>
      Sign in as Dev User
    </button>
  );
}

function EmailLoginForm() {
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const result = await api.emailLogin(email, password);
      await login(result.token);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      {error && <div className="error">{error}</div>}
      <form onSubmit={handleSubmit} className="login-form">
        <div className="form-group">
          <label>Email</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        <div className="form-group">
          <label>Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        <button type="submit" className="btn btn-primary" disabled={loading} style={{ width: '100%' }}>
          {loading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>
      <p className="login-link">
        Don't have an account? <Link to="/register">Register</Link>
      </p>
    </>
  );
}

export default function Login({ devMode }: { devMode: boolean }) {
  return (
    <div className="login-container">
      <div className="login-card">
        <h1>PUC Redyr Golf Scoring</h1>
        <p>Sign in to manage tournament scores</p>
        {devMode ? (
          <DevLogin />
        ) : (
          <>
            <EmailLoginForm />
            <div className="login-divider">
              <span>or</span>
            </div>
            <Suspense fallback={<div>Loading...</div>}>
              <GoogleLoginButton />
            </Suspense>
          </>
        )}
      </div>
    </div>
  );
}
