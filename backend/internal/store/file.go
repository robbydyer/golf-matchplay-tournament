package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"scoring-backend/internal/models"
	"strings"
	"sync"
	"time"
)

// FileStore persists each tournament as a JSON file on disk.
// Files are stored as {dir}/{tournament-id}.json.
type FileStore struct {
	mu  sync.RWMutex
	dir string
}

func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory %s: %w", dir, err)
	}
	return &FileStore{dir: dir}, nil
}

func (f *FileStore) path(id string) string {
	return filepath.Join(f.dir, id+".json")
}

func (f *FileStore) readTournament(id string) (*models.Tournament, error) {
	data, err := os.ReadFile(f.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("tournament %s not found", id)
		}
		return nil, fmt.Errorf("reading tournament %s: %w", id, err)
	}

	var t models.Tournament
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("decoding tournament %s: %w", id, err)
	}
	// Normalize: ensure all matches have HoleResults initialized (handles old data files)
	for i := range t.Rounds {
		for j := range t.Rounds[i].Matches {
			if t.Rounds[i].Matches[j].HoleResults == nil {
				t.Rounds[i].Matches[j].HoleResults = make([]string, 18)
			}
		}
	}
	return &t, nil
}

func (f *FileStore) writeTournament(t *models.Tournament) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding tournament %s: %w", t.ID, err)
	}

	// Write to temp file then rename for atomic writes
	tmp := f.path(t.ID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing tournament %s: %w", t.ID, err)
	}
	if err := os.Rename(tmp, f.path(t.ID)); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming tournament file %s: %w", t.ID, err)
	}
	return nil
}

func (f *FileStore) CreateTournament(_ context.Context, t *models.Tournament) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, err := os.Stat(f.path(t.ID)); err == nil {
		return fmt.Errorf("tournament %s already exists", t.ID)
	}

	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	return f.writeTournament(t)
}

func (f *FileStore) GetTournament(_ context.Context, id string) (*models.Tournament, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.readTournament(id)
}

func (f *FileStore) UpdateTournament(_ context.Context, t *models.Tournament) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, err := os.Stat(f.path(t.ID)); os.IsNotExist(err) {
		return fmt.Errorf("tournament %s not found", t.ID)
	}

	t.UpdatedAt = time.Now()
	return f.writeTournament(t)
}

func (f *FileStore) ListTournaments(_ context.Context) ([]*models.Tournament, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return nil, fmt.Errorf("listing data directory: %w", err)
	}

	tournaments := make([]*models.Tournament, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5] // strip .json
		t, err := f.readTournament(id)
		if err != nil {
			continue // skip corrupt files
		}
		tournaments = append(tournaments, t)
	}
	return tournaments, nil
}

func (f *FileStore) DeleteTournament(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	p := f.path(id)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return fmt.Errorf("tournament %s not found", id)
	}

	if err := os.Remove(p); err != nil {
		return fmt.Errorf("deleting tournament %s: %w", id, err)
	}
	return nil
}

func (f *FileStore) UpdateMatchResult(_ context.Context, tournamentID string, roundNumber int, matchID string, result models.MatchResult, score string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, err := f.readTournament(tournamentID)
	if err != nil {
		return err
	}

	for i := range t.Rounds {
		if t.Rounds[i].Number != roundNumber {
			continue
		}
		for j := range t.Rounds[i].Matches {
			if t.Rounds[i].Matches[j].ID == matchID {
				t.Rounds[i].Matches[j].Result = result
				t.Rounds[i].Matches[j].Score = score
				t.UpdatedAt = time.Now()
				return f.writeTournament(t)
			}
		}
		return fmt.Errorf("match %s not found in round %d", matchID, roundNumber)
	}

	return fmt.Errorf("round %d not found", roundNumber)
}

func (f *FileStore) SetRoundPairings(_ context.Context, tournamentID string, roundNumber int, matches []models.Match) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, err := f.readTournament(tournamentID)
	if err != nil {
		return err
	}

	for i := range t.Rounds {
		if t.Rounds[i].Number == roundNumber {
			t.Rounds[i].Matches = matches
			t.UpdatedAt = time.Now()
			return f.writeTournament(t)
		}
	}

	return fmt.Errorf("round %d not found", roundNumber)
}

func (f *FileStore) usersPath() string {
	return filepath.Join(f.dir, "_users.json")
}

func (f *FileStore) RegisterUser(_ context.Context, user *models.RegisteredUser) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	users := make(map[string]*models.RegisteredUser)
	data, err := os.ReadFile(f.usersPath())
	if err == nil {
		json.Unmarshal(data, &users)
	}
	users[user.Email] = user
	out, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding users: %w", err)
	}
	tmp := f.usersPath() + ".tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return fmt.Errorf("writing users: %w", err)
	}
	if err := os.Rename(tmp, f.usersPath()); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming users file: %w", err)
	}
	return nil
}

func (f *FileStore) ListRegisteredUsers(_ context.Context) ([]*models.RegisteredUser, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := os.ReadFile(f.usersPath())
	if err != nil {
		if os.IsNotExist(err) {
			return make([]*models.RegisteredUser, 0), nil
		}
		return nil, fmt.Errorf("reading users: %w", err)
	}

	var users map[string]*models.RegisteredUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("decoding users: %w", err)
	}

	result := make([]*models.RegisteredUser, 0, len(users))
	for _, u := range users {
		result = append(result, u)
	}
	return result, nil
}

func (f *FileStore) LinkPlayer(_ context.Context, tournamentID string, playerID string, email string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, err := f.readTournament(tournamentID)
	if err != nil {
		return err
	}

	for ti := range t.Teams {
		for pi := range t.Teams[ti].Players {
			if t.Teams[ti].Players[pi].ID == playerID {
				t.Teams[ti].Players[pi].UserEmail = email
				t.UpdatedAt = time.Now()
				return f.writeTournament(t)
			}
		}
	}

	return fmt.Errorf("player %s not found", playerID)
}

func (f *FileStore) UpdateHoleResult(_ context.Context, tournamentID string, roundNumber int, matchID string, hole int, result string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	t, err := f.readTournament(tournamentID)
	if err != nil {
		return err
	}

	for i := range t.Rounds {
		if t.Rounds[i].Number != roundNumber {
			continue
		}
		for j := range t.Rounds[i].Matches {
			if t.Rounds[i].Matches[j].ID == matchID {
				match := &t.Rounds[i].Matches[j]
				if match.HoleResults == nil {
					match.HoleResults = make([]string, 18)
				}
				match.HoleResults[hole] = result
				// Backfill any earlier empty holes as halved
				for h := 0; h < hole; h++ {
					if match.HoleResults[h] == "" {
						match.HoleResults[h] = "halved"
					}
				}
				match.Result, match.Score = models.CalculateMatchPlayResult(match.HoleResults, t.Teams[0].Name, t.Teams[1].Name)
				t.UpdatedAt = time.Now()
				return f.writeTournament(t)
			}
		}
		return fmt.Errorf("match %s not found in round %d", matchID, roundNumber)
	}

	return fmt.Errorf("round %d not found", roundNumber)
}
