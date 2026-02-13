import { useState } from 'react';
import { Tournament } from '../types';
import * as api from '../api/client';

interface Props {
  tournament: Tournament;
  onUpdate: () => void;
  isAdmin: boolean;
}

export default function TeamSetup({ tournament, onUpdate, isAdmin }: Props) {
  const [teams, setTeams] = useState(() =>
    tournament.teams.map((t) => ({
      name: t.name,
      players: t.players.length > 0
        ? t.players.map((p) => p.name)
        : Array(8).fill(''),
    }))
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const updatePlayerName = (teamIdx: number, playerIdx: number, name: string) => {
    const updated = [...teams];
    updated[teamIdx] = {
      ...updated[teamIdx],
      players: [...updated[teamIdx].players],
    };
    updated[teamIdx].players[playerIdx] = name;
    setTeams(updated);
  };

  const updateTeamName = (teamIdx: number, name: string) => {
    const updated = [...teams];
    updated[teamIdx] = { ...updated[teamIdx], name };
    setTeams(updated);
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      await api.updateTournament(tournament.id, {
        teams: [
          { name: teams[0].name, players: teams[0].players.map((n) => ({ name: n })) },
          { name: teams[1].name, players: teams[1].players.map((n) => ({ name: n })) },
        ],
      });
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="team-setup">
      {error && <div className="error">{error}</div>}

      <div className="teams-grid">
        {teams.map((team, ti) => (
          <div key={ti} className="card team-card">
            <div className="form-group">
              <label>Team Name</label>
              <input
                value={team.name}
                onChange={(e) => updateTeamName(ti, e.target.value)}
                readOnly={!isAdmin}
              />
            </div>
            <h4>Players</h4>
            {team.players.map((name, pi) => (
              <div key={pi} className="form-group player-input">
                <label>#{pi + 1}</label>
                <input
                  value={name}
                  onChange={(e) => updatePlayerName(ti, pi, e.target.value)}
                  placeholder={`Player ${pi + 1}`}
                  readOnly={!isAdmin}
                />
              </div>
            ))}
          </div>
        ))}
      </div>

      {isAdmin && (
        <div className="form-actions">
          <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : 'Save Teams'}
          </button>
        </div>
      )}
    </div>
  );
}
