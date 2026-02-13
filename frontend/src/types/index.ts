export type RoundType = 'lauderdale' | 'foursome' | 'fourball' | 'singles';
export type MatchResult = 'pending' | 'team1' | 'team2' | 'tie';

export interface Player {
  id: string;
  name: string;
  teamId: string;
  userEmail?: string;
}

export interface RegisteredUser {
  email: string;
  name: string;
  picture: string;
}

export interface Team {
  id: string;
  name: string;
  players: Player[];
}

export type HoleResult = '' | 'team1' | 'team2' | 'halved';

export interface Match {
  id: string;
  roundNumber: number;
  team1Players: string[];
  team2Players: string[];
  result: MatchResult;
  score: string;
  holeResults: HoleResult[] | null;
}

export interface Round {
  number: number;
  name: string;
  type: RoundType;
  pointsPerMatch: number;
  matches: Match[];
}

export interface Tournament {
  id: string;
  name: string;
  teams: [Team, Team];
  rounds: Round[];
  createdAt: string;
  updatedAt: string;
}

export interface Scoreboard {
  team1Name: string;
  team2Name: string;
  team1Total: number;
  team2Total: number;
  roundScores: RoundScore[];
}

export interface RoundScore {
  roundNumber: number;
  roundName: string;
  team1Points: number;
  team2Points: number;
  pointsPerMatch: number;
  matchesPlayed: number;
  totalMatches: number;
}

export interface User {
  email: string;
  name: string;
  picture: string;
  isAdmin: boolean;
}
