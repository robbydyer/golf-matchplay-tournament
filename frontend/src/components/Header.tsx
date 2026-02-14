import { useAuth } from '../contexts/AuthContext';

export default function Header() {
  const { user, logout } = useAuth();

  return (
    <header className="app-header">
      <h1>PUC Redyr Golf Scoring</h1>
      {user && (
        <div className="user-info">
          {user.picture ? (
            <img src={user.picture} alt={user.name} className="avatar" />
          ) : (
            <div className="avatar avatar-placeholder">
              {user.name?.[0]?.toUpperCase() || '?'}
            </div>
          )}
          <span>{user.name}</span>
          <button onClick={logout} className="btn btn-sm">
            Sign Out
          </button>
        </div>
      )}
    </header>
  );
}
