import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Header() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  return (
    <header className="app-header">
      <h1>PUC Redyr Golf Scoring</h1>
      {user && (
        <div className="user-info">
          {user.isAdmin && (
            <button onClick={() => navigate('/admin/users')} className="btn btn-sm">
              Manage Users
            </button>
          )}
          <div className="avatar avatar-placeholder">
            {user.name?.[0]?.toUpperCase() || '?'}
          </div>
          <span>{user.name}</span>
          <button onClick={logout} className="btn btn-sm">
            Sign Out
          </button>
        </div>
      )}
    </header>
  );
}
