import { useRef, useLayoutEffect } from 'react';
import { Scoreboard, Tournament, Player } from '../types';

// Parse hex color to rgba with given opacity
function hexToRgba(hex: string, alpha: number): string {
  const h = hex.replace('#', '');
  const r = parseInt(h.substring(0, 2), 16);
  const g = parseInt(h.substring(2, 4), 16);
  const b = parseInt(h.substring(4, 6), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

interface Props {
  scoreboard: Scoreboard;
  tournament: Tournament;
  fullscreen?: boolean;
}

export default function ScoreboardView({ scoreboard, tournament, fullscreen }: Props) {
  const maxPoints = scoreboard.roundScores.reduce(
    (sum, rs) => sum + rs.totalMatches * rs.pointsPerMatch,
    0
  );

  const team1Pct = maxPoints > 0 ? (scoreboard.team1Total / maxPoints) * 100 : 50;
  const team2Pct = maxPoints > 0 ? (scoreboard.team2Total / maxPoints) * 100 : 50;

  const playerMap = new Map<string, Player>();
  [...tournament.teams[0].players, ...tournament.teams[1].players].forEach((p) =>
    playerMap.set(p.id, p)
  );
  const playerName = (id: string) => playerMap.get(id)?.name || id;

  const roundScoreMap = new Map(
    scoreboard.roundScores.map((rs) => [rs.roundNumber, rs])
  );

  const team1Color = tournament.teams[0].color || '#1a3a6b';
  const team2Color = tournament.teams[1].color || '#8b1a1a';

  const hasAnyMatches = tournament.rounds.some((r) => r.matches.length > 0);

  const roundsRef = useRef<HTMLDivElement>(null);
  const rosterRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    if (!fullscreen) return;
    const el = roundsRef.current;
    if (!el) return;

    const apply = () => {
      // Reset first so measurements are unaffected by previous zoom
      (el.style as any).zoom = '';

      // Each grid cell has a fixed fr height — measure per-section overflow
      // since the grid's own scrollHeight won't capture it
      const sections = el.querySelectorAll<HTMLElement>('.round-matches-section');
      let maxRatio = 1;
      sections.forEach((s) => {
        const available = s.clientHeight;
        const needed = s.scrollHeight;
        if (available > 0 && needed > available) {
          maxRatio = Math.max(maxRatio, needed / available);
        }
      });

      if (maxRatio > 1) {
        (el.style as any).zoom = String(1 / maxRatio);
      }
    };

    apply();
    const ro = new ResizeObserver(apply);
    ro.observe(el);
    return () => ro.disconnect();
  }, [fullscreen, scoreboard, tournament]);

  useLayoutEffect(() => {
    if (hasAnyMatches) return;
    const el = rosterRef.current;
    if (!el) return;

    const apply = () => {
      (el.style as any).zoom = '';
      const top = el.getBoundingClientRect().top;
      const available = window.innerHeight - top - 16; // 16px bottom breathing room
      const needed = el.scrollHeight;
      if (available > 0 && needed > 0) {
        (el.style as any).zoom = String(available / needed);
      }
    };

    apply();
    const ro = new ResizeObserver(apply);
    ro.observe(el);
    return () => ro.disconnect();
  }, [hasAnyMatches, scoreboard, tournament]);

  return (
    <div className="scoreboard">
      <div className="score-total">
        {tournament.teams[0].logo && (
          <img className="team-logo team-logo-left" src={`/${tournament.teams[0].logo}`} alt={scoreboard.team1Name} />
        )}
        <div className="team-score">
          <span className="team-name" style={{ color: team1Color }}>{scoreboard.team1Name}</span>
          <span className="score" style={{ color: team1Color }}>{scoreboard.team1Total}</span>
        </div>
        <div className="score-divider">-</div>
        <div className="team-score">
          <span className="score" style={{ color: team2Color }}>{scoreboard.team2Total}</span>
          <span className="team-name" style={{ color: team2Color }}>{scoreboard.team2Name}</span>
        </div>
        {tournament.teams[1].logo && (
          <img className="team-logo team-logo-right" src={`/${tournament.teams[1].logo}`} alt={scoreboard.team2Name} />
        )}
      </div>

      {maxPoints > 0 && (
        <div className="score-win-target">
          {maxPoints / 2 + 0.5} points needed to win
        </div>
      )}

      <div className="score-bar">
        <div className="bar-team1" style={{ width: `${team1Pct}%`, background: team1Color }} />
        <div className="bar-team2" style={{ width: `${team2Pct}%`, background: team2Color }} />
      </div>

      {!hasAnyMatches && (
        <div className="scoreboard-rosters" ref={rosterRef}>
          {tournament.teams.map((team, i) => {
            const color = i === 0 ? team1Color : team2Color;
            return (
              <div key={team.id} className="roster-card">
                <h3 style={{ color }}>{team.name}</h3>
                {team.players.length === 0 ? (
                  <p className="empty">No players yet.</p>
                ) : (
                  <ol className="roster-list">
                    {team.players.map((p) => (
                      <li key={p.id}>{p.name}</li>
                    ))}
                  </ol>
                )}
              </div>
            );
          })}
        </div>
      )}

      <div className="rounds-grid" ref={roundsRef}>
        {tournament.rounds.map((round) => {
          if (round.matches.length === 0) return null;
          const rs = roundScoreMap.get(round.number);
          return (
            <div key={round.number} className={`round-matches-section${round.number === 5 ? ' round-last' : ''}`}>
              <div className="round-header">
                <h4>{round.name}</h4>
                {rs && (
                  <div className="round-summary">
                    <span className="round-pts" style={{ color: team1Color }}>{rs.team1Points}</span>
                    <span className="round-pts-divider">-</span>
                    <span className="round-pts" style={{ color: team2Color }}>{rs.team2Points}</span>
                    <span className="round-meta">
                      {rs.pointsPerMatch} pt/match
                    </span>
                  </div>
                )}
              </div>
              <table className="matches-table">
                <thead>
                  <tr>
                    <th>{tournament.teams[0].name}</th>
                    <th className="result-col">Result</th>
                    <th>{tournament.teams[1].name}</th>
                  </tr>
                </thead>
                <tbody>
                  {round.matches.map((match) => {
                    const t1Names = match.team1Players.map(playerName).join(' & ');
                    const t2Names = match.team2Players.map(playerName).join(' & ');

                    let resultLabel: string;
                    let resultClass: string;
                    const scoreText = match.score || '';
                    if (match.result === 'team1') {
                      resultLabel = `${tournament.teams[0].name} win`;
                      resultClass = 'result-team1';
                    } else if (match.result === 'team2') {
                      resultLabel = `${tournament.teams[1].name} win`;
                      resultClass = 'result-team2';
                    } else if (match.result === 'tie') {
                      resultLabel = 'Halved';
                      resultClass = 'result-tie';
                    } else {
                      resultLabel = '-';
                      resultClass = 'result-pending';
                    }

                    const rowBg = match.result === 'team1'
                      ? hexToRgba(team1Color, 0.1)
                      : match.result === 'team2'
                        ? hexToRgba(team2Color, 0.1)
                        : match.result === 'tie'
                          ? 'rgba(133, 100, 4, 0.06)'
                          : undefined;

                    const t1Border = match.result === 'team1' ? `3px solid ${team1Color}` : undefined;
                    const t2Border = match.result === 'team2' ? `3px solid ${team2Color}` : undefined;
                    const resultColor = match.result === 'team1' ? team1Color
                      : match.result === 'team2' ? team2Color
                        : undefined;

                    return (
                      <tr key={match.id} style={rowBg ? { background: rowBg } : undefined}>
                        <td className={`match-team-cell ${match.result === 'team1' ? 'winner-cell' : ''}`}
                          style={t1Border ? { borderLeft: t1Border } : undefined}>
                          {t1Names}
                        </td>
                        <td className={`result-cell ${resultClass}`}
                          style={resultColor ? { color: resultColor } : undefined}>
                          <div>{resultLabel}</div>
                          {scoreText && <div className="result-score">{scoreText}</div>}
                        </td>
                        <td className={`match-team-cell ${match.result === 'team2' ? 'winner-cell' : ''}`}
                          style={t2Border ? { borderRight: t2Border } : undefined}>
                          {t2Names}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          );
        })}
      </div>
    </div>
  );
}
