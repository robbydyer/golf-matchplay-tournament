import { Scoreboard, Tournament, Player } from '../types';

interface Props {
  scoreboard: Scoreboard;
  tournament: Tournament;
}

export default function ScoreboardView({ scoreboard, tournament }: Props) {
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

  return (
    <div className="scoreboard">
      <div className="score-total">
        <div className="team-score team1">
          <span className="team-name">{scoreboard.team1Name}</span>
          <span className="score">{scoreboard.team1Total}</span>
        </div>
        <div className="score-divider">-</div>
        <div className="team-score team2">
          <span className="team-name">{scoreboard.team2Name}</span>
          <span className="score">{scoreboard.team2Total}</span>
        </div>
      </div>

      <div className="score-bar">
        <div className="bar-team1" style={{ width: `${team1Pct}%` }} />
        <div className="bar-team2" style={{ width: `${team2Pct}%` }} />
      </div>

      <div className="rounds-grid">
        {tournament.rounds.map((round) => {
          if (round.matches.length === 0) return null;
          const rs = roundScoreMap.get(round.number);
          return (
            <div key={round.number} className={`round-matches-section${round.number === 5 ? ' round-last' : ''}`}>
              <div className="round-header">
                <h4>{round.name}</h4>
                {rs && (
                  <div className="round-summary">
                    <span className="round-pts team1">{rs.team1Points}</span>
                    <span className="round-pts-divider">-</span>
                    <span className="round-pts team2">{rs.team2Points}</span>
                    <span className="round-meta">
                      {rs.pointsPerMatch} pt/match &middot; {rs.matchesPlayed}/{rs.totalMatches} played
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

                    return (
                      <tr key={match.id}>
                        <td className={`match-team-cell ${match.result === 'team1' ? 'winner-cell' : ''}`}>
                          {t1Names}
                        </td>
                        <td className={`result-cell ${resultClass}`}>
                          <div>{resultLabel}</div>
                          {scoreText && <div className="result-score">{scoreText}</div>}
                        </td>
                        <td className={`match-team-cell ${match.result === 'team2' ? 'winner-cell' : ''}`}>
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
