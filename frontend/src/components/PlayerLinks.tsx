import { useState, useEffect } from 'react';
import { Tournament, RegisteredUser } from '../types';
import * as api from '../api/client';

interface Props {
  tournament: Tournament;
  onUpdate: () => void;
}

export default function PlayerLinks({ tournament, onUpdate }: Props) {
  const [users, setUsers] = useState<RegisteredUser[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    api.listUsers().then(setUsers).catch((e) => setError(e.message));
  }, []);

  const handleLink = async (playerId: string, email: string) => {
    setError('');
    try {
      await api.linkPlayer(tournament.id, playerId, email);
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    }
  };

  return (
    <div className="player-links">
      {error && <div className="error">{error}</div>}

      <p className="player-links-info">
        Link tournament players to user accounts. Only users who have signed in appear in the list.
      </p>

      <div className="teams-grid">
        {tournament.teams.map((team, ti) => (
          <div key={ti} className="card team-card">
            <h4>{team.name}</h4>
            {team.players.length === 0 ? (
              <p className="empty">No players yet.</p>
            ) : (
              <div className="link-list">
                {team.players.map((player) => (
                  <div key={player.id} className="link-row">
                    <span className="link-player-name">{player.name}</span>
                    <select
                      className="link-select"
                      value={player.userEmail || ''}
                      onChange={(e) => handleLink(player.id, e.target.value)}
                    >
                      <option value="">-- No user linked --</option>
                      {users.map((u) => (
                        <option key={u.email} value={u.email}>
                          {u.name} ({u.email})
                        </option>
                      ))}
                    </select>
                  </div>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
