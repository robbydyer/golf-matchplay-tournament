# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Golf tournament match-play scoring app for the PUC Redyr Cup. Two teams of 8 players compete across 5 rounds with different formats (Lauderdale, Foursome, Four-Ball, Singles). Go backend with React/TypeScript frontend, containerized with Docker.

## Build & Run Commands

```bash
# Local development (requires Docker)
docker compose build
docker compose up -d

# Rebuild a single service
docker compose build backend   # or frontend
docker compose up -d backend

# Backend only (Go 1.22)
cd backend && go build ./cmd/server

# View logs
docker compose logs -f backend
docker compose logs -f frontend
```

The user's machine cannot run `npm install` locally (esbuild native binary issue). All frontend builds must happen inside Docker. Add dependencies by editing `frontend/package.json` directly.

Frontend dev server runs on `http://localhost:5173`, backend on `http://localhost:8080`. Vite proxies `/api` to the backend.

## Environment

Set in `.env` (loaded by docker-compose):
- `DEV_MODE=true` — bypasses Google OAuth, all users are admin
- `STORE_BACKEND=file` — persistence mode (`memory`, `file`, `firestore`)
- `ADMIN_EMAILS=a@b.com,c@d.com` — comma-separated admin emails
- `VITE_GOOGLE_CLIENT_ID` — required for Google OAuth when `DEV_MODE=false`
- `JWT_SECRET` — HMAC secret for signing local auth tokens (required for production)
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM` — SMTP config for verification emails
- `APP_URL` — frontend URL for verification email links (defaults to `CORS_ORIGIN`)

## Architecture

### Backend (`backend/`)

Go standard library HTTP server (Go 1.22 method-based routing). No framework.

- **Entry**: `cmd/server/main.go` — env config, store init, middleware chain (CORS → Auth → routes)
- **Handlers**: `internal/handlers/handlers.go` — all REST endpoints, route registration
- **Models**: `internal/models/models.go` — Tournament, Match, Player, Round structs; match play scoring logic (`CalculateMatchPlayResult`)
- **Auth**: `internal/auth/auth.go` — Google OAuth token verification via `oauth2.googleapis.com/tokeninfo`, local email/password auth with HMAC-signed tokens, `RequireAdmin` middleware, dev mode bypass
- **Email**: `internal/email/email.go` — SMTP email sending for verification
- **Store**: `internal/store/store.go` — interface; `memory.go` (in-RAM), `file.go` (JSON files with atomic writes), `firestore.go` (stub)

Store pattern: all mutations read-modify-write the full tournament. FileStore uses mutex + temp-file-then-rename for atomicity. Files stored as `data/{tournament-id}.json`, user registry as `data/_users.json`, local users as `data/_local_users.json`.

### Frontend (`frontend/`)

React 18 + TypeScript + Vite. Client-side routing via `react-router-dom` v6.

- **Routes** (defined in `App.tsx`): `/` (list), `/tournament/:id/:tab` (detail with tabs), `/register`, `/verify`
- **Auth**: `contexts/AuthContext.tsx` — Google OAuth, email/password, or dev mode; token in localStorage, refreshes via `/api/me` on mount
- **API**: `api/client.ts` — typed fetch wrapper, bearer token auth; `publicFetch` for unauthenticated auth endpoints
- **Key components**: `TournamentView.tsx` (tab router), `RoundView.tsx` (pairings + hole-by-hole scoring), `ScoreboardView.tsx`, `TeamSetup.tsx`, `PlayerLinks.tsx`, `Register.tsx`, `VerifyEmail.tsx`

### Key Data Flow

`HoleResults` is a `map[int]string` (Go) / `Record<string, HoleResult>` (TS) keyed by 1-based hole number. Values: `"team1"`, `"team2"`, `"halved"`. Empty holes are absent from the map. The backend auto-calculates match result and score from hole results (clinch detection, "X & Y" / "X UP" / "A/S" formatting) and auto-backfills earlier holes as halved.

### Authentication

Two auth methods, both producing Bearer tokens for API access:
- **Google OAuth**: Frontend gets access token via `@react-oauth/google`, backend validates via Google's tokeninfo endpoint
- **Email/Password**: Register at `/api/auth/register` (bcrypt hashed, requires email verification), login at `/api/auth/login` returns HMAC-signed token (`local.<payload>.<sig>`)

Public endpoints (`/api/auth/*`) bypass the auth middleware. The middleware detects token type by the `local.` prefix.

When SMTP is not configured, verification tokens are logged to stdout for manual use.

### Authorization Model

- **Admin** (email in `ADMIN_EMAILS`): full control — create tournaments, set pairings, edit results, link players
- **Linked player** (player's `userEmail` matches logged-in user): can edit hole results for their own matches
- **Any authenticated user**: can view everything including hole-by-hole scores

### Backward Compatibility

`Match.UnmarshalJSON` handles migration from the old `[]string` array format for `holeResults` to the current `map[int]string` format. Old data files are migrated transparently on read.

## Production Deployment

Uses `frontend/Dockerfile.prod` (multi-stage: node build → nginx). The `nginx.conf` has SPA fallback (`try_files $uri $uri/ /index.html`). Set `CORS_ORIGIN` to the frontend's origin (not the backend's).

## No Tests

There are currently no test files in either backend or frontend.
