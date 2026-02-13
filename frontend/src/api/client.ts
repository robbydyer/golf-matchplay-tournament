import { Tournament, Scoreboard, MatchResult, HoleResult, User } from '../types';

const API_BASE = (import.meta.env.VITE_API_URL || '') + '/api';

function getToken(): string | null {
  return localStorage.getItem('access_token');
}

async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  if (!token) {
    throw new Error('Not authenticated');
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || `Request failed: ${res.status}`);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

export async function getMe(): Promise<User> {
  return apiFetch<User>('/me');
}

export async function listTournaments(): Promise<Tournament[]> {
  return apiFetch<Tournament[]>('/tournaments');
}

export async function createTournament(name: string, team1Name: string, team2Name: string): Promise<Tournament> {
  return apiFetch<Tournament>('/tournaments', {
    method: 'POST',
    body: JSON.stringify({ name, team1Name, team2Name }),
  });
}

export async function getTournament(id: string): Promise<Tournament> {
  return apiFetch<Tournament>(`/tournaments/${id}`);
}

export async function updateTournament(
  id: string,
  data: {
    name?: string;
    teams?: [{ name: string; players: { name: string }[] }, { name: string; players: { name: string }[] }];
  }
): Promise<Tournament> {
  return apiFetch<Tournament>(`/tournaments/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteTournament(id: string): Promise<void> {
  return apiFetch<void>(`/tournaments/${id}`, { method: 'DELETE' });
}

export async function getScoreboard(id: string): Promise<Scoreboard> {
  return apiFetch<Scoreboard>(`/tournaments/${id}/scoreboard`);
}

export async function setPairings(
  tournamentId: string,
  roundNumber: number,
  matches: { team1Players: string[]; team2Players: string[] }[]
): Promise<Tournament> {
  return apiFetch<Tournament>(`/tournaments/${tournamentId}/rounds/${roundNumber}/pairings`, {
    method: 'PUT',
    body: JSON.stringify({ matches }),
  });
}

export async function updateMatchResult(
  tournamentId: string,
  roundNumber: number,
  matchId: string,
  result: MatchResult,
  score: string
): Promise<Tournament> {
  return apiFetch<Tournament>(`/tournaments/${tournamentId}/rounds/${roundNumber}/matches/${matchId}`, {
    method: 'PUT',
    body: JSON.stringify({ result, score }),
  });
}

export async function updateHoleResult(
  tournamentId: string,
  roundNumber: number,
  matchId: string,
  hole: number,
  result: HoleResult
): Promise<Tournament> {
  return apiFetch<Tournament>(`/tournaments/${tournamentId}/rounds/${roundNumber}/matches/${matchId}/holes/${hole}`, {
    method: 'PUT',
    body: JSON.stringify({ result }),
  });
}
