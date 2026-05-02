import { useState, useEffect } from 'react';
import { Tournament, Player, PlayerRanking } from '../types';
import * as api from '../api/client';
import { useAuth } from '../contexts/AuthContext';

interface Props {
  tournament: Tournament;
}

export default function RankingView({ tournament }: Props) {
  const { user } = useAuth();
  const isAdmin = user?.isAdmin ?? false;

  // Find which team and player the current user is linked to
  const linkedTeamIndex = tournament.teams.findIndex((team) =>
    team.players.some((p) => p.userEmail?.toLowerCase() === user?.email?.toLowerCase())
  );
  const myTeam = linkedTeamIndex >= 0 ? tournament.teams[linkedTeamIndex] : null;

  const [order, setOrder] = useState<Player[]>([]);
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [allRankings, setAllRankings] = useState<PlayerRanking[]>([]);
  const [rankingsLoading, setRankingsLoading] = useState(false);

  // Initialize order from existing ranking or default team order
  useEffect(() => {
    if (!myTeam) return;
    const load = async () => {
      try {
        const rankings = await api.getRankings(tournament.id);
        const mine = rankings.find((r) => r.submittedBy.toLowerCase() === user?.email?.toLowerCase());
        if (mine) {
          const playerMap = new Map(myTeam.players.map((p) => [p.id, p]));
          const ordered = mine.playerIds
            .map((id) => playerMap.get(id))
            .filter((p): p is Player => !!p);
          // Add any players not in the ranking (e.g. roster changed)
          const inRanking = new Set(mine.playerIds);
          myTeam.players.filter((p) => !inRanking.has(p.id)).forEach((p) => ordered.push(p));
          setOrder(ordered);
        } else {
          setOrder([...myTeam.players]);
        }
      } catch {
        setOrder([...myTeam.players]);
      }
    };
    load();
  }, [tournament.id, myTeam, user?.email]);

  // Load all rankings for admin view
  useEffect(() => {
    if (!isAdmin) return;
    setRankingsLoading(true);
    api.getRankings(tournament.id)
      .then(setAllRankings)
      .catch(() => {})
      .finally(() => setRankingsLoading(false));
  }, [tournament.id, isAdmin]);

  const moveUp = (index: number) => {
    if (index === 0) return;
    setOrder((prev) => {
      const next = [...prev];
      [next[index - 1], next[index]] = [next[index], next[index - 1]];
      return next;
    });
    setSaved(false);
  };

  const moveDown = (index: number) => {
    if (index === order.length - 1) return;
    setOrder((prev) => {
      const next = [...prev];
      [next[index], next[index + 1]] = [next[index + 1], next[index]];
      return next;
    });
    setSaved(false);
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      await api.submitRanking(tournament.id, order.map((p) => p.id));
      setSaved(true);
      // Reload admin rankings if applicable
      if (isAdmin) {
        const rankings = await api.getRankings(tournament.id);
        setAllRankings(rankings);
      }
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  // Compute aggregate rankings per team for admin view
  const computeAggregates = (teamIndex: number) => {
    const team = tournament.teams[teamIndex];
    const teamEmails = new Set(team.players.map((p) => p.userEmail?.toLowerCase()).filter(Boolean));
    const teamRankings = allRankings.filter((r) => teamEmails.has(r.submittedBy.toLowerCase()));

    return team.players.map((player) => {
      const ranks: number[] = [];
      teamRankings.forEach((r) => {
        const pos = r.playerIds.indexOf(player.id);
        if (pos >= 0) ranks.push(pos + 1);
      });
      const avg = ranks.length > 0 ? ranks.reduce((a, b) => a + b, 0) / ranks.length : null;
      return { player, avg, responses: ranks.length };
    }).sort((a, b) => {
      if (a.avg === null && b.avg === null) return 0;
      if (a.avg === null) return 1;
      if (b.avg === null) return -1;
      return a.avg - b.avg;
    });
  };

  const teamColor = (index: number) => tournament.teams[index].color || (index === 0 ? '#1a3a6b' : '#8b1a1a');

  return (
    <div className="ranking-view">
      {myTeam ? (
        <div className="card manage-card">
          <h3>Your Team Ranking — <span style={{ color: teamColor(linkedTeamIndex) }}>{myTeam.name}</span></h3>
          <p style={{ color: '#555', marginBottom: '1rem', fontSize: '0.9rem' }}>
            Drag or use the arrows to rank your teammates from best (1st) to last.
          </p>
          {tournament.rankingsLocked && !isAdmin && (
            <div className="info-banner" style={{ marginBottom: '1rem' }}>Rankings are locked. Your submission cannot be changed.</div>
          )}
          {error && <div className="error">{error}</div>}
          <ol className="ranking-list">
            {order.map((player, i) => (
              <li key={player.id} className="ranking-item">
                <span className="rank-number">{i + 1}</span>
                <span className="rank-name">{player.name}</span>
                <div className="rank-actions">
                  <button
                    className="btn btn-sm"
                    onClick={() => moveUp(i)}
                    disabled={i === 0}
                    title="Move up"
                  >↑</button>
                  <button
                    className="btn btn-sm"
                    onClick={() => moveDown(i)}
                    disabled={i === order.length - 1}
                    title="Move down"
                  >↓</button>
                </div>
              </li>
            ))}
          </ol>
          <div className="form-actions" style={{ marginTop: '1rem' }}>
            <button
              className="btn btn-primary"
              onClick={handleSave}
              disabled={saving || (!!tournament.rankingsLocked && !isAdmin)}
            >
              {saving ? 'Saving...' : saved ? 'Saved ✓' : 'Save Ranking'}
            </button>
          </div>
        </div>
      ) : !isAdmin ? (
        <p className="empty">You are not linked to a player on this tournament. Ask an admin to link your account.</p>
      ) : null}

      {isAdmin && (
        <div className="ranking-admin">
          <h3>Aggregate Rankings</h3>
          {rankingsLoading ? (
            <div className="loading"><div className="spinner" /><div>Loading...</div></div>
          ) : (
            <div className="ranking-teams-grid">
              {tournament.teams.map((team, ti) => {
                const agg = computeAggregates(ti);
                const total = new Set(
                  allRankings
                    .filter((r) => team.players.some((p) => p.userEmail?.toLowerCase() === r.submittedBy.toLowerCase()))
                    .map((r) => r.submittedBy)
                ).size;
                return (
                  <div key={team.id} className="card">
                    <h4 style={{ color: teamColor(ti), marginBottom: '0.75rem' }}>
                      {team.name}
                      <span style={{ fontWeight: 'normal', fontSize: '0.85rem', marginLeft: '0.5rem', color: '#666' }}>
                        ({total} of {team.players.length} responded)
                      </span>
                    </h4>
                    {team.players.length === 0 ? (
                      <p className="empty">No players on this team.</p>
                    ) : (
                      <table className="ranking-table">
                        <thead>
                          <tr>
                            <th>Rank</th>
                            <th>Player</th>
                            <th>Avg</th>
                          </tr>
                        </thead>
                        <tbody>
                          {agg.map((row, i) => (
                            <tr key={row.player.id}>
                              <td className="rank-pos">{i + 1}</td>
                              <td>{row.player.name}</td>
                              <td className="rank-avg">
                                {row.avg !== null ? row.avg.toFixed(1) : '—'}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
