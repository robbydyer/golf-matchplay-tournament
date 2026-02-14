import { useState, useEffect } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import * as api from '../api/client';

export default function VerifyEmail() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');
  const [status, setStatus] = useState<'verifying' | 'success' | 'error'>('verifying');
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!token) {
      setStatus('error');
      setMessage('No verification token provided.');
      return;
    }

    api.verifyEmail(token)
      .then((result) => {
        setStatus('success');
        setMessage(result.message);
      })
      .catch((err) => {
        setStatus('error');
        setMessage(err.message);
      });
  }, [token]);

  return (
    <div className="login-container">
      <div className="login-card">
        {status === 'verifying' && (
          <>
            <h1>Verifying Email</h1>
            <p>Please wait...</p>
          </>
        )}
        {status === 'success' && (
          <>
            <h1>Email Verified</h1>
            <p>{message}</p>
            <Link to="/" className="btn btn-primary" style={{ display: 'inline-block', marginTop: '16px' }}>
              Sign In
            </Link>
          </>
        )}
        {status === 'error' && (
          <>
            <h1>Verification Failed</h1>
            <p>{message}</p>
            <Link to="/" className="btn" style={{ display: 'inline-block', marginTop: '16px' }}>
              Back to Login
            </Link>
          </>
        )}
      </div>
    </div>
  );
}
