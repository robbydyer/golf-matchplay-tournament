import { useAuth } from '../contexts/AuthContext';

export default function Header() {
  const { user, logout } = useAuth();

  return (
    <header className="app-header">
      <h1>PUC Redyr Golf Scoring</h1>
      {user && (
        <div className="user-info">
          <img src={user.picture} alt={user.name} className="avatar" />
          <span>{user.name}</span>
          <button onClick={logout} className="btn btn-sm">
            Sign Out
          </button>
        </div>
      )}
    </header>
  );
}
