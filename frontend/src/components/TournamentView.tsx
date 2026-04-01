import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { Tournament, Scoreboard } from '../types';
import * as api from '../api/client';
import ScoreboardView from './ScoreboardView';
import TeamSetup from './TeamSetup';
import RoundView from './RoundView';
import PlayerLinks from './PlayerLinks';
import ManageView from './ManageView';
import Header from './Header';
import { useAuth } from '../contexts/AuthContext';

export default function TournamentView() {
  const { id: tournamentId, tab } = useParams<{ id: string; tab: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { user } = useAuth();
  const isAdmin = user?.isAdmin ?? false;
  const [tournament, setTournament] = useState<Tournament | null>(null);
  const [scoreboard, setScoreboard] = useState<Scoreboard | null>(null);
  const [error, setError] = useState('');
  const [fullscreen, setFullscreen] = useState(() => searchParams.get('fullscreen') === 'true');
  const [editingName, setEditingName] = useState(false);
  const [nameInput, setNameInput] = useState('');

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

  // Escape key exits fullscreen
  useEffect(() => {
    if (!fullscreen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setFullscreen(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [fullscreen]);

  // Apply tournament colors to CSS custom properties
  useEffect(() => {
    if (!tournament) return;
    const root = document.documentElement;
    if (tournament.headerColor) {
      root.style.setProperty('--color-primary', tournament.headerColor);
    } else {
      root.style.removeProperty('--color-primary');
    }
    if (tournament.bgColor) {
      root.style.setProperty('--color-bg', tournament.bgColor);
    } else {
      root.style.removeProperty('--color-bg');
    }
    return () => {
      root.style.removeProperty('--color-primary');
      root.style.removeProperty('--color-bg');
    };
  }, [tournament]);

  if (!tournament || !scoreboard) {
    return <div className="loading"><div className="spinner" /><div>Loading...</div></div>;
  }

  const teamsReady = tournament.teams[0].players.length === 8 && tournament.teams[1].players.length === 8;

  const saveName = async () => {
    const trimmed = nameInput.trim();
    if (!trimmed || trimmed === tournament.name) {
      setEditingName(false);
      return;
    }
    try {
      await api.updateTournament(tournament.id, { name: trimmed });
      setEditingName(false);
      load();
    } catch (e: any) {
      setError(e.message);
    }
  };

  const navTo = (path: string) => navigate(`/tournament/${tournamentId}/${path}`, { replace: true });

  if (fullscreen && activeTab === 'scoreboard') {
    return (
      <div className="scoreboard-fullscreen">
        <Header />
        <ScoreboardView scoreboard={scoreboard} tournament={tournament} fullscreen />
        <button className="btn fullscreen-exit" onClick={() => setFullscreen(false)}>
          Exit Fullscreen
        </button>
      </div>
    );
  }

  return (
    <div className="tournament-view">
      <div className="tournament-header">
        <button className="btn" onClick={() => navigate('/')}>&larr; Back</button>
        {editingName ? (
          <input
            className="tournament-name-input"
            value={nameInput}
            onChange={(e) => setNameInput(e.target.value)}
            onBlur={saveName}
            onKeyDown={(e) => {
              if (e.key === 'Enter') saveName();
              if (e.key === 'Escape') setEditingName(false);
            }}
            autoFocus
          />
        ) : (
          <h2
            className={isAdmin ? 'editable-name' : ''}
            onClick={() => { if (isAdmin) { setNameInput(tournament.name); setEditingName(true); } }}
            title={isAdmin ? 'Click to edit tournament name' : undefined}
          >
            {tournament.name}
          </h2>
        )}
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
          <>
            <button
              className={`tab ${activeTab === 'links' ? 'active' : ''}`}
              onClick={() => navTo('links')}
            >
              Player Links
            </button>
            <button
              className={`tab ${activeTab === 'manage' ? 'active' : ''}`}
              onClick={() => navTo('manage')}
            >
              Manage
            </button>
          </>
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
          <>
            <button className="btn btn-fullscreen" onClick={() => setFullscreen(true)}>
              Fullscreen
            </button>
            <ScoreboardView scoreboard={scoreboard} tournament={tournament} />
          </>
        )}
        {activeTab === 'teams' && (
          <TeamSetup tournament={tournament} onUpdate={load} isAdmin={isAdmin} />
        )}
        {activeTab === 'links' && isAdmin && (
          <PlayerLinks tournament={tournament} onUpdate={load} />
        )}
        {activeTab === 'manage' && isAdmin && (
          <ManageView tournament={tournament} onUpdate={load} />
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
