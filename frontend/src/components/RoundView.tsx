import { useState } from 'react';
import { Tournament, Match, Player, MatchResult, HoleResult } from '../types';
import * as api from '../api/client';

interface Props {
  tournament: Tournament;
  roundNumber: number;
  onUpdate: () => void;
  teamsReady: boolean;
  isAdmin: boolean;
}

export default function RoundView({ tournament, roundNumber, onUpdate, teamsReady, isAdmin }: Props) {
  const round = tournament.rounds.find((r) => r.number === roundNumber)!;
  const team1 = tournament.teams[0];
  const team2 = tournament.teams[1];
  const isSingles = round.type === 'singles';

  const [settingUp, setSettingUp] = useState(false);
  const [pairings, setPairings] = useState<{ t1: string[]; t2: string[] }[]>([]);
  const [scores, setScores] = useState<Record<string, string>>({});
  const [expandedHoles, setExpandedHoles] = useState<Record<string, boolean>>({});
  const [error, setError] = useState('');

  const playerMap = new Map<string, Player>();
  [...team1.players, ...team2.players].forEach((p) => playerMap.set(p.id, p));

  const getPlayerName = (id: string) => playerMap.get(id)?.name || id;

  const matchCount = isSingles ? 8 : 4;
  const playersPerSide = isSingles ? 1 : 2;

  const initPairings = () => {
    const newPairings = Array.from({ length: matchCount }, () => ({
      t1: Array(playersPerSide).fill(''),
      t2: Array(playersPerSide).fill(''),
    }));
    setPairings(newPairings);
    setSettingUp(true);
  };

  const editPairings = () => {
    const existing = round.matches.map((m) => ({
      t1: [
        ...m.team1Players,
        ...Array(Math.max(0, playersPerSide - m.team1Players.length)).fill(''),
      ].slice(0, playersPerSide),
      t2: [
        ...m.team2Players,
        ...Array(Math.max(0, playersPerSide - m.team2Players.length)).fill(''),
      ].slice(0, playersPerSide),
    }));
    // Pad to expected match count if fewer matches exist
    while (existing.length < matchCount) {
      existing.push({
        t1: Array(playersPerSide).fill(''),
        t2: Array(playersPerSide).fill(''),
      });
    }
    setPairings(existing);
    setSettingUp(true);
  };

  const updatePairing = (matchIdx: number, team: 't1' | 't2', playerIdx: number, playerId: string) => {
    const updated = pairings.map((p, i) => {
      if (i !== matchIdx) return p;
      const side = [...p[team]];
      side[playerIdx] = playerId;
      return { ...p, [team]: side };
    });
    setPairings(updated);
  };

  const savePairings = async () => {
    setError('');
    const matches = pairings.map((p) => ({
      team1Players: p.t1.filter(Boolean),
      team2Players: p.t2.filter(Boolean),
    }));

    // Validate all slots filled
    const allFilled = matches.every(
      (m) => m.team1Players.length === playersPerSide && m.team2Players.length === playersPerSide
    );
    if (!allFilled) {
      setError('All player slots must be filled');
      return;
    }

    try {
      await api.setPairings(tournament.id, roundNumber, matches);
      setSettingUp(false);
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    }
  };

  const getScore = (matchId: string) => scores[matchId] ?? '';

  const handleScoreChange = (matchId: string, value: string) => {
    setScores((prev) => ({ ...prev, [matchId]: value }));
  };

  const handleResult = async (match: Match, result: MatchResult) => {
    try {
      const score = result === 'pending' ? '' : (scores[match.id] ?? match.score ?? '');
      await api.updateMatchResult(tournament.id, roundNumber, match.id, result, score);
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    }
  };

  const handleScoreSave = async (match: Match) => {
    try {
      const score = scores[match.id] ?? match.score ?? '';
      await api.updateMatchResult(tournament.id, roundNumber, match.id, match.result, score);
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    }
  };

  const toggleHoles = (matchId: string) => {
    setExpandedHoles((prev) => ({ ...prev, [matchId]: !prev[matchId] }));
  };

  const handleHoleResult = async (match: Match, hole: number, result: HoleResult) => {
    try {
      const current = match.holeResults?.[hole - 1] || '';
      // Toggle: if already set to this result, clear it
      const newResult: HoleResult = current === result ? '' : result;
      await api.updateHoleResult(tournament.id, roundNumber, match.id, hole, newResult);
      onUpdate();
    } catch (e: any) {
      setError(e.message);
    }
  };

  const getHoleStatus = (match: Match) => {
    if (!match.holeResults) return { t1: 0, t2: 0, halved: 0, played: 0 };
    let t1 = 0, t2 = 0, halved = 0;
    for (const r of match.holeResults) {
      if (r === 'team1') t1++;
      else if (r === 'team2') t2++;
      else if (r === 'halved') halved++;
    }
    return { t1, t2, halved, played: t1 + t2 + halved };
  };

  if (!teamsReady) {
    return (
      <div className="round-view">
        <h3>{round.name}</h3>
        <p className="empty">Set up both teams with 8 players before configuring rounds.</p>
      </div>
    );
  }

  // Available players for dropdowns (filter already-used ones)
  const usedT1 = new Set(pairings.flatMap((p) => p.t1).filter(Boolean));
  const usedT2 = new Set(pairings.flatMap((p) => p.t2).filter(Boolean));

  return (
    <div className="round-view">
      <div className="section-header">
        <div>
          <h3>{round.name}</h3>
          <span className="badge">{round.pointsPerMatch} pt/match</span>
        </div>
        {!settingUp && isAdmin && (
          round.matches.length === 0 ? (
            <button className="btn btn-primary" onClick={initPairings}>
              Set Up Pairings
            </button>
          ) : (
            <button className="btn" onClick={editPairings}>
              Edit Pairings
            </button>
          )
        )}
      </div>

      {error && <div className="error">{error}</div>}

      {settingUp && (
        <div className="pairings-setup">
          {pairings.map((pairing, mi) => (
            <div key={mi} className="card pairing-card">
              <h4>Match {mi + 1}</h4>
              <div className="pairing-sides">
                <div className="pairing-team">
                  <label>{team1.name}</label>
                  {Array.from({ length: playersPerSide }, (_, pi) => (
                    <div key={pi} className="pairing-select">
                      <select
                        value={pairing.t1[pi]}
                        onChange={(e) => updatePairing(mi, 't1', pi, e.target.value)}
                      >
                        <option value="">Select player...</option>
                        {team1.players
                          .filter((p) => !usedT1.has(p.id) || pairing.t1[pi] === p.id)
                          .map((p) => (
                            <option key={p.id} value={p.id}>{p.name}</option>
                          ))}
                      </select>
                      {pairing.t1[pi] && (
                        <button
                          className="btn-clear"
                          onClick={() => updatePairing(mi, 't1', pi, '')}
                          title="Clear selection"
                        >&times;</button>
                      )}
                    </div>
                  ))}
                </div>
                <div className="vs">vs</div>
                <div className="pairing-team">
                  <label>{team2.name}</label>
                  {Array.from({ length: playersPerSide }, (_, pi) => (
                    <div key={pi} className="pairing-select">
                      <select
                        value={pairing.t2[pi]}
                        onChange={(e) => updatePairing(mi, 't2', pi, e.target.value)}
                      >
                        <option value="">Select player...</option>
                        {team2.players
                          .filter((p) => !usedT2.has(p.id) || pairing.t2[pi] === p.id)
                          .map((p) => (
                            <option key={p.id} value={p.id}>{p.name}</option>
                          ))}
                      </select>
                      {pairing.t2[pi] && (
                        <button
                          className="btn-clear"
                          onClick={() => updatePairing(mi, 't2', pi, '')}
                          title="Clear selection"
                        >&times;</button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ))}
          <div className="form-actions">
            <button className="btn btn-primary" onClick={savePairings}>Save Pairings</button>
            <button className="btn" onClick={() => setSettingUp(false)}>Cancel</button>
          </div>
        </div>
      )}

      {round.matches.length > 0 && !settingUp && (
        <div className="matches">
          {round.matches.map((match, mi) => (
            <div key={match.id} className={`card match-card ${match.result !== 'pending' ? 'completed' : ''}`}>
              <div className="match-header">Match {mi + 1}</div>
              <div className="match-players">
                <div className={`match-side ${match.result === 'team1' ? 'winner' : ''}`}>
                  {match.team1Players.map((id) => (
                    <span key={id} className="player-name">{getPlayerName(id)}</span>
                  ))}
                </div>
                <div className="vs">vs</div>
                <div className={`match-side ${match.result === 'team2' ? 'winner' : ''}`}>
                  {match.team2Players.map((id) => (
                    <span key={id} className="player-name">{getPlayerName(id)}</span>
                  ))}
                </div>
              </div>
              {isAdmin ? (
                <div className="match-score-row">
                  <label>Score</label>
                  <input
                    className="score-input"
                    value={scores[match.id] ?? match.score ?? ''}
                    onChange={(e) => handleScoreChange(match.id, e.target.value)}
                    onBlur={() => {
                      if (match.result !== 'pending' && (scores[match.id] ?? '') !== '' && scores[match.id] !== match.score) {
                        handleScoreSave(match);
                      }
                    }}
                    placeholder="e.g. 2 & 1, 1 UP, A/S"
                  />
                </div>
              ) : match.score ? (
                <div className="match-score-row">
                  <label>Score</label>
                  <span className="score-display">{match.score}</span>
                </div>
              ) : null}
              {isAdmin && (
                <div className="match-actions">
                  <button
                    className={`btn btn-sm ${match.result === 'team1' ? 'btn-winner' : ''}`}
                    onClick={() => handleResult(match, 'team1')}
                  >
                    {team1.name} wins
                  </button>
                  <button
                    className={`btn btn-sm ${match.result === 'tie' ? 'btn-tie' : ''}`}
                    onClick={() => handleResult(match, 'tie')}
                  >
                    Tie
                  </button>
                  <button
                    className={`btn btn-sm ${match.result === 'team2' ? 'btn-winner' : ''}`}
                    onClick={() => handleResult(match, 'team2')}
                  >
                    {team2.name} wins
                  </button>
                  {match.result !== 'pending' && (
                    <button
                      className="btn btn-sm"
                      onClick={() => handleResult(match, 'pending')}
                    >
                      Reset
                    </button>
                  )}
                </div>
              )}
              <div className="hole-by-hole-section">
                <button
                  className="btn btn-sm hole-toggle"
                  onClick={() => toggleHoles(match.id)}
                >
                  {expandedHoles[match.id] ? 'Hide Holes' : 'Hole by Hole'}
                  {(() => {
                    const s = getHoleStatus(match);
                    return s.played > 0 ? ` (${s.t1}-${s.t2}-${s.halved})` : '';
                  })()}
                </button>
                {expandedHoles[match.id] && (
                  <div className="holes-grid">
                    {Array.from({ length: 18 }, (_, i) => {
                      const holeNum = i + 1;
                      const current = (match.holeResults?.[i] || '') as HoleResult;
                      return (
                        <div key={holeNum} className="hole-cell">
                          <span className="hole-number">{holeNum}</span>
                          <div className="hole-buttons">
                            <button
                              className={`hole-btn hole-t1 ${current === 'team1' ? 'active' : ''}`}
                              onClick={() => handleHoleResult(match, holeNum, 'team1')}
                              title={`${team1.name} wins hole ${holeNum}`}
                            >
                              {team1.name.substring(0, 3)}
                            </button>
                            <button
                              className={`hole-btn hole-halved ${current === 'halved' ? 'active' : ''}`}
                              onClick={() => handleHoleResult(match, holeNum, 'halved')}
                              title={`Hole ${holeNum} halved`}
                            >
                              &#189;
                            </button>
                            <button
                              className={`hole-btn hole-t2 ${current === 'team2' ? 'active' : ''}`}
                              onClick={() => handleHoleResult(match, holeNum, 'team2')}
                              title={`${team2.name} wins hole ${holeNum}`}
                            >
                              {team2.name.substring(0, 3)}
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
