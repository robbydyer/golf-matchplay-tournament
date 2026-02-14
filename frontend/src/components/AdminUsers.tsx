import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { LocalUserInfo } from '../types';
import * as api from '../api/client';

export default function AdminUsers() {
  const navigate = useNavigate();
  const [users, setUsers] = useState<LocalUserInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const fetchUsers = async () => {
    try {
      const data = await api.listLocalUsers();
      setUsers(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleConfirm = async (email: string) => {
    setActionLoading(email);
    try {
      await api.confirmUser(email);
      await fetchUsers();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setActionLoading(null);
    }
  };

  const handleReject = async (email: string) => {
    if (!confirm(`Remove user ${email}? This cannot be undone.`)) return;
    setActionLoading(email);
    try {
      await api.rejectUser(email);
      await fetchUsers();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setActionLoading(null);
    }
  };

  if (loading) return <div className="loading">Loading users...</div>;

  const pending = users.filter(u => u.emailVerified && !u.confirmed);
  const active = users.filter(u => u.emailVerified && u.confirmed);
  const unverified = users.filter(u => !u.emailVerified);

  return (
    <div>
      <div className="section-header">
        <h2>Manage Users</h2>
        <button className="btn btn-sm" onClick={() => navigate('/')}>Back</button>
      </div>

      {error && <div className="error">{error}</div>}

      <div className="admin-section">
        <h3>Pending Approval ({pending.length})</h3>
        {pending.length === 0 ? (
          <p className="admin-empty">No users pending approval.</p>
        ) : (
          <div className="admin-user-list">
            {pending.map(u => (
              <div key={u.email} className="admin-user-row">
                <div className="admin-user-info">
                  <span className="admin-user-name">{u.name}</span>
                  <span className="admin-user-email">{u.email}</span>
                </div>
                <div className="admin-user-actions">
                  <button
                    className="btn btn-primary btn-sm"
                    onClick={() => handleConfirm(u.email)}
                    disabled={actionLoading === u.email}
                  >
                    {actionLoading === u.email ? '...' : 'Approve'}
                  </button>
                  <button
                    className="btn btn-sm btn-danger"
                    onClick={() => handleReject(u.email)}
                    disabled={actionLoading === u.email}
                  >
                    Reject
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="admin-section">
        <h3>Active Users ({active.length})</h3>
        {active.length === 0 ? (
          <p className="admin-empty">No active users.</p>
        ) : (
          <div className="admin-user-list">
            {active.map(u => (
              <div key={u.email} className="admin-user-row">
                <div className="admin-user-info">
                  <span className="admin-user-name">{u.name}</span>
                  <span className="admin-user-email">{u.email}</span>
                </div>
                <div className="admin-user-actions">
                  <button
                    className="btn btn-sm btn-danger"
                    onClick={() => handleReject(u.email)}
                    disabled={actionLoading === u.email}
                  >
                    Remove
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {unverified.length > 0 && (
        <div className="admin-section">
          <h3>Awaiting Email Verification ({unverified.length})</h3>
          <div className="admin-user-list">
            {unverified.map(u => (
              <div key={u.email} className="admin-user-row">
                <div className="admin-user-info">
                  <span className="admin-user-name">{u.name}</span>
                  <span className="admin-user-email">{u.email}</span>
                </div>
                <div className="admin-user-actions">
                  <button
                    className="btn btn-sm btn-danger"
                    onClick={() => handleReject(u.email)}
                    disabled={actionLoading === u.email}
                  >
                    Remove
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
