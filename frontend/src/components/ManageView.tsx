import { useState, useEffect } from 'react';
import { Tournament, LocalUserInfo } from '../types';
import * as api from '../api/client';

interface Props {
  tournament: Tournament;
  onUpdate: () => void;
}

const DEFAULT_HEADER = '#1C4932';
const DEFAULT_BG = '#f5f5f0';

export default function ManageView({ tournament, onUpdate }: Props) {
  const [headerColor, setHeaderColor] = useState(tournament.headerColor || DEFAULT_HEADER);
  const [bgColor, setBgColor] = useState(tournament.bgColor || DEFAULT_BG);
  const [saving, setSaving] = useState(false);
  const [locking, setLocking] = useState(false);
  const [combiningRounds, setCombiningRounds] = useState(false);
  const [error, setError] = useState('');

  const [users, setUsers] = useState<LocalUserInfo[]>([]);
  const [usersLoading, setUsersLoading] = useState(true);
  const [deletingEmail, setDeletingEmail] = useState<string | null>(null);
  const [approvingEmail, setApprovingEmail] = useState<string | null>(null);
  const [enablingEmail, setEnablingEmail] = useState<string | null>(null);

  const loadUsers = async () => {
    try {
      const data = await api.listLocalUsers();
      setUsers(data);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setUsersLoading(false);
    }
  };

  useEffect(() => { loadUsers(); }, []);

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      await api.updateTournament(tournament.id, { headerColor, bgColor });
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setHeaderColor(DEFAULT_HEADER);
    setBgColor(DEFAULT_BG);
  };

  const handleToggleCombineRounds = async () => {
    setCombiningRounds(true);
    setError('');
    try {
      await api.combineRounds(tournament.id, !tournament.combineRounds23);
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setCombiningRounds(false);
    }
  };

  const handleToggleLock = async () => {
    setLocking(true);
    setError('');
    try {
      await api.lockTournament(tournament.id, !tournament.locked);
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLocking(false);
    }
  };

  const handleApprove = async (email: string) => {
    setApprovingEmail(email);
    try {
      await api.confirmUser(email);
      await loadUsers();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setApprovingEmail(null);
    }
  };

  const handleDelete = async (email: string) => {
    if (!confirm(`Disable user ${email}?`)) return;
    setDeletingEmail(email);
    try {
      await api.rejectUser(email);
      await loadUsers();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setDeletingEmail(null);
    }
  };

  const handleEnable = async (email: string) => {
    setEnablingEmail(email);
    try {
      await api.enableUser(email);
      await loadUsers();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setEnablingEmail(null);
    }
  };

  return (
    <div className="manage-view">
      {error && <div className="error">{error}</div>}

      <div className="card manage-card">
        <h3>Appearance</h3>

        <div className="manage-colors">
          <div className="form-group">
            <label>Header Bar Color</label>
            <div className="team-color-input">
              <input
                type="color"
                value={headerColor}
                onChange={(e) => setHeaderColor(e.target.value)}
              />
              <span>{headerColor}</span>
            </div>
          </div>

          <div className="form-group">
            <label>Background Color</label>
            <div className="team-color-input">
              <input
                type="color"
                value={bgColor}
                onChange={(e) => setBgColor(e.target.value)}
              />
              <span>{bgColor}</span>
            </div>
          </div>
        </div>

        <div className="form-actions">
          <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : 'Save'}
          </button>
          <button className="btn" onClick={handleReset}>
            Reset to Defaults
          </button>
        </div>
      </div>

      <div className="card manage-card">
        <h3>Round Format</h3>
        <p style={{ marginBottom: '1rem', color: '#555' }}>
          {tournament.combineRounds23
            ? 'Rounds 2 & 3 are combined into a single 18-hole Foursome round. Pairings and scoring are entered under the R2-3 tab.'
            : 'Rounds 2 and 3 are played as separate 9-hole Foursome sessions (Friday PM and Saturday AM).'}
        </p>
        <div className="form-actions">
          <button
            className="btn btn-primary"
            onClick={handleToggleCombineRounds}
            disabled={combiningRounds}
          >
            {combiningRounds ? '...' : tournament.combineRounds23 ? 'Split into Separate Rounds' : 'Combine Rounds 2 & 3'}
          </button>
        </div>
      </div>

      <div className="card manage-card">
        <h3>Tournament Lock</h3>
        <p style={{ marginBottom: '1rem', color: '#555' }}>
          {tournament.locked
            ? 'Tournament is locked. Players cannot edit hole results. Admins can still make changes.'
            : 'Tournament is unlocked. Linked players can enter hole results for their matches.'}
        </p>
        <div className="form-actions">
          <button
            className={`btn ${tournament.locked ? 'btn-primary' : 'btn-danger'}`}
            onClick={handleToggleLock}
            disabled={locking}
          >
            {locking ? '...' : tournament.locked ? 'Unlock Tournament' : 'Lock Tournament'}
          </button>
        </div>
      </div>

      <div className="card manage-card">
        <h3>Users</h3>
        {usersLoading ? (
          <div className="loading"><div className="spinner" /><div>Loading...</div></div>
        ) : users.length === 0 ? (
          <p className="empty">No users.</p>
        ) : (
          <table className="manage-users-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.email}>
                  <td>{u.name}</td>
                  <td>{u.email}</td>
                  <td>
                    {u.disabled ? 'Disabled' : !u.emailVerified ? 'Unverified' : !u.confirmed ? 'Pending approval' : 'Active'}
                  </td>
                  <td>
                    <div className="manage-users-actions">
                      {u.emailVerified && !u.confirmed && !u.disabled && (
                        <button
                          className="btn btn-primary btn-sm"
                          onClick={() => handleApprove(u.email)}
                          disabled={approvingEmail === u.email}
                        >
                          {approvingEmail === u.email ? '...' : 'Approve'}
                        </button>
                      )}
                      {u.disabled ? (
                        <button
                          className="btn btn-primary btn-sm"
                          onClick={() => handleEnable(u.email)}
                          disabled={enablingEmail === u.email}
                        >
                          {enablingEmail === u.email ? '...' : 'Reactivate'}
                        </button>
                      ) : (
                        <button
                          className="btn btn-danger btn-sm"
                          onClick={() => handleDelete(u.email)}
                          disabled={deletingEmail === u.email}
                        >
                          {deletingEmail === u.email ? '...' : 'Disable'}
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
