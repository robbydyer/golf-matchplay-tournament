import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Tournament, Scoreboard } from '../types';
import * as api from '../api/client';
import ScoreboardView from './ScoreboardView';
import TeamSetup from './TeamSetup';
import RoundView from './RoundView';
import PlayerLinks from './PlayerLinks';
import { useAuth } from '../contexts/AuthContext';

export default function TournamentView() {
  const { id: tournamentId, tab } = useParams<{ id: string; tab: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.isAdmin ?? false;
  const [tournament, setTournament] = useState<Tournament | null>(null);
  const [scoreboard, setScoreboard] = useState<Scoreboard | null>(null);
  const [error, setError] = useState('');

  // Parse tab param into activeTab and activeRound
  let activeTab: string;
  let activeRound = 1;
  if (tab && tab.startsWith('round')) {
    activeTab = 'round';
    const roundNum = parseInt(tab.replace('round', ''), 10);
    if (roundNum >= 1 && roundNum <= 5) {
      activeRound = roundNum;
    }
  } else {
    activeTab = tab || 'scoreboard';
  }

  const load = useCallback(async () => {
    if (!tournamentId) return;
    try {
      const [t, sb] = await Promise.all([
        api.getTournament(tournamentId),
        api.getScoreboard(tournamentId),
      ]);
      setTournament(t);
      setScoreboard(sb);
    } catch (e: any) {
      setError(e.message);
    }
  }, [tournamentId]);

  useEffect(() => {
    load();
    const interval = setInterval(load, 10000);
    return () => clearInterval(interval);
  }, [load]);

  if (!tournament || !scoreboard) {
    return <div className="loading">Loading...</div>;
  }

  const teamsReady = tournament.teams[0].players.length === 8 && tournament.teams[1].players.length === 8;

  const navTo = (path: string) => navigate(`/tournament/${tournamentId}/${path}`, { replace: true });

  return (
    <div className="tournament-view">
      <div className="tournament-header">
        <button className="btn" onClick={() => navigate('/')}>&larr; Back</button>
        <h2>{tournament.name}</h2>
      </div>

      {error && <div className="error">{error}</div>}

      <nav className="tabs">
        <button
          className={`tab ${activeTab === 'scoreboard' ? 'active' : ''}`}
          onClick={() => navTo('scoreboard')}
        >
          Scoreboard
        </button>
        <button
          className={`tab ${activeTab === 'teams' ? 'active' : ''}`}
          onClick={() => navTo('teams')}
        >
          Teams
        </button>
        {isAdmin && (
          <button
            className={`tab ${activeTab === 'links' ? 'active' : ''}`}
            onClick={() => navTo('links')}
          >
            Player Links
          </button>
        )}
        {[1, 2, 3, 4, 5].map((r) => (
          <button
            key={r}
            className={`tab ${activeTab === 'round' && activeRound === r ? 'active' : ''}`}
            onClick={() => navTo(`round${r}`)}
          >
            R{r}
          </button>
        ))}
      </nav>

      <div className="tab-content">
        {activeTab === 'scoreboard' && (
          <ScoreboardView scoreboard={scoreboard} tournament={tournament} />
        )}
        {activeTab === 'teams' && (
          <TeamSetup tournament={tournament} onUpdate={load} isAdmin={isAdmin} />
        )}
        {activeTab === 'links' && isAdmin && (
          <PlayerLinks tournament={tournament} onUpdate={load} />
        )}
        {activeTab === 'round' && (
          <RoundView
            tournament={tournament}
            roundNumber={activeRound}
            onUpdate={load}
            teamsReady={teamsReady}
            isAdmin={isAdmin}
          />
        )}
      </div>
    </div>
  );
}
