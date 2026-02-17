import { Tournament, Scoreboard, MatchResult, HoleResult, User, RegisteredUser, LocalUserInfo } from '../types';

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

// Public fetch for unauthenticated endpoints (register, login, verify)
async function publicFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || `Request failed: ${res.status}`);
  }

  return res.json();
}

// --- Public auth endpoints ---

export async function register(email: string, name: string, password: string): Promise<{ message: string }> {
  return publicFetch('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, name, password }),
  });
}

export async function emailLogin(email: string, password: string): Promise<{ token: string }> {
  return publicFetch('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function verifyEmail(token: string): Promise<{ message: string }> {
  return publicFetch('/auth/verify', {
    method: 'POST',
    body: JSON.stringify({ token }),
  });
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

export async function updateRoundName(
  tournamentId: string,
  roundNumber: number,
  name: string
): Promise<Tournament> {
  return apiFetch<Tournament>(`/tournaments/${tournamentId}/rounds/${roundNumber}/name`, {
    method: 'PUT',
    body: JSON.stringify({ name }),
  });
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

export async function listUsers(): Promise<RegisteredUser[]> {
  return apiFetch<RegisteredUser[]>('/users');
}

export async function linkPlayer(
  tournamentId: string,
  playerId: string,
  email: string
): Promise<Tournament> {
  return apiFetch<Tournament>(`/tournaments/${tournamentId}/players/${playerId}/link`, {
    method: 'PUT',
    body: JSON.stringify({ email }),
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

// --- Admin user management ---

export async function listLocalUsers(): Promise<LocalUserInfo[]> {
  return apiFetch<LocalUserInfo[]>('/admin/users');
}

export async function confirmUser(email: string): Promise<{ message: string }> {
  return apiFetch('/admin/users/confirm', {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
}

export async function rejectUser(email: string): Promise<{ message: string }> {
  return apiFetch('/admin/users/reject', {
    method: 'POST',
    body: JSON.stringify({ email }),
  });
}
