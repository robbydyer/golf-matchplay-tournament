import { useState, useEffect, useCallback } from 'react';
import { Tournament, Scoreboard } from '../types';
import * as api from '../api/client';
import ScoreboardView from './ScoreboardView';
import TeamSetup from './TeamSetup';
import RoundView from './RoundView';
import PlayerLinks from './PlayerLinks';
import { useAuth } from '../contexts/AuthContext';

interface Props {
  tournamentId: string;
  onBack: () => void;
}

type Tab = 'scoreboard' | 'teams' | 'round' | 'links';

export default function TournamentView({ tournamentId, onBack }: Props) {
  const { user } = useAuth();
  const isAdmin = user?.isAdmin ?? false;
  const [tournament, setTournament] = useState<Tournament | null>(null);
  const [scoreboard, setScoreboard] = useState<Scoreboard | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('scoreboard');
  const [activeRound, setActiveRound] = useState(1);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
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
  }, [load]);

  if (!tournament || !scoreboard) {
    return <div className="loading">Loading...</div>;
  }

  const teamsReady = tournament.teams[0].players.length === 8 && tournament.teams[1].players.length === 8;

  return (
    <div className="tournament-view">
      <div className="tournament-header">
        <button className="btn" onClick={onBack}>&larr; Back</button>
        <h2>{tournament.name}</h2>
      </div>

      {error && <div className="error">{error}</div>}

      <nav className="tabs">
        <button
          className={`tab ${activeTab === 'scoreboard' ? 'active' : ''}`}
          onClick={() => setActiveTab('scoreboard')}
        >
          Scoreboard
        </button>
        <button
          className={`tab ${activeTab === 'teams' ? 'active' : ''}`}
          onClick={() => setActiveTab('teams')}
        >
          Teams
        </button>
        {isAdmin && (
          <button
            className={`tab ${activeTab === 'links' ? 'active' : ''}`}
            onClick={() => setActiveTab('links')}
          >
            Player Links
          </button>
        )}
        {[1, 2, 3, 4, 5].map((r) => (
          <button
            key={r}
            className={`tab ${activeTab === 'round' && activeRound === r ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('round');
              setActiveRound(r);
            }}
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
