import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Tournament } from '../types';
import * as api from '../api/client';
import { useAuth } from '../contexts/AuthContext';

export default function TournamentList() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.isAdmin ?? false;
  const [tournaments, setTournaments] = useState<Tournament[]>([]);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({ name: '', team1Name: '', team2Name: '' });
  const [error, setError] = useState('');

  const load = async () => {
    try {
      const list = await api.listTournaments();
      setTournaments(list || []);
    } catch (e: any) {
      setError(e.message);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleCreate = async () => {
    if (!form.name || !form.team1Name || !form.team2Name) {
      setError('All fields are required');
      return;
    }
    try {
      const t = await api.createTournament(form.name, form.team1Name, form.team2Name);
      setCreating(false);
      setForm({ name: '', team1Name: '', team2Name: '' });
      navigate(`/tournament/${t.id}/scoreboard`);
    } catch (e: any) {
      setError(e.message);
    }
  };

  return (
    <div className="tournament-list">
      <div className="section-header">
        <h2>Tournaments</h2>
        {isAdmin && (
          <button className="btn btn-primary" onClick={() => setCreating(true)}>
            New Tournament
          </button>
        )}
      </div>

      {error && <div className="error">{error}</div>}

      {creating && (
        <div className="create-form card">
          <h3>Create Tournament</h3>
          <div className="form-group">
            <label>Tournament Name</label>
            <input
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="e.g. PUC Redyr Cup 2026"
            />
          </div>
          <div className="form-row">
            <div className="form-group">
              <label>Team 1 Name</label>
              <input
                value={form.team1Name}
                onChange={(e) => setForm({ ...form, team1Name: e.target.value })}
                placeholder="e.g. Team Alpha"
              />
            </div>
            <div className="form-group">
              <label>Team 2 Name</label>
              <input
                value={form.team2Name}
                onChange={(e) => setForm({ ...form, team2Name: e.target.value })}
                placeholder="e.g. Team Bravo"
              />
            </div>
          </div>
          <div className="form-actions">
            <button className="btn btn-primary" onClick={handleCreate}>Create</button>
            <button className="btn" onClick={() => setCreating(false)}>Cancel</button>
          </div>
        </div>
      )}

      {tournaments.length === 0 && !creating ? (
        <p className="empty">No tournaments yet. Create one to get started.</p>
      ) : (
        <div className="card-grid">
          {tournaments.map((t) => (
            <div key={t.id} className="card clickable" onClick={() => navigate(`/tournament/${t.id}/scoreboard`)}>
              <h3>{t.name}</h3>
              <p>
                {t.teams[0].name} vs {t.teams[1].name}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
