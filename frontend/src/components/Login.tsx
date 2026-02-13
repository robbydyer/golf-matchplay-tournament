import { lazy, Suspense } from 'react';
import { useAuth } from '../contexts/AuthContext';

const GoogleLoginButton = lazy(() => import('./GoogleLoginButton'));

function DevLogin() {
  const { devLogin } = useAuth();

  return (
    <button className="btn btn-primary" onClick={devLogin}>
      Sign in as Dev User
    </button>
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
          <Suspense fallback={<div>Loading...</div>}>
            <GoogleLoginButton />
          </Suspense>
        )}
      </div>
    </div>
  );
}
