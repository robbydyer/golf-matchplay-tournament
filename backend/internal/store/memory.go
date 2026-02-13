package store

import (
	"context"
	"fmt"
	"scoring-backend/internal/models"
	"sync"
	"time"
)

type MemoryStore struct {
	mu          sync.RWMutex
	tournaments map[string]*models.Tournament
	users       map[string]*models.RegisteredUser
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tournaments: make(map[string]*models.Tournament),
		users:       make(map[string]*models.RegisteredUser),
	}
}

func (m *MemoryStore) CreateTournament(_ context.Context, t *models.Tournament) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tournaments[t.ID]; exists {
		return fmt.Errorf("tournament %s already exists", t.ID)
	}

	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	// Deep copy to avoid external mutation
	copied := *t
	m.tournaments[t.ID] = &copied
	return nil
}

func (m *MemoryStore) GetTournament(_ context.Context, id string) (*models.Tournament, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.tournaments[id]
	if !ok {
		return nil, fmt.Errorf("tournament %s not found", id)
	}

	copied := *t
	return &copied, nil
}

func (m *MemoryStore) UpdateTournament(_ context.Context, t *models.Tournament) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tournaments[t.ID]; !ok {
		return fmt.Errorf("tournament %s not found", t.ID)
	}

	t.UpdatedAt = time.Now()
	copied := *t
	m.tournaments[t.ID] = &copied
	return nil
}

func (m *MemoryStore) ListTournaments(_ context.Context) ([]*models.Tournament, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.Tournament, 0, len(m.tournaments))
	for _, t := range m.tournaments {
		copied := *t
		result = append(result, &copied)
	}
	return result, nil
}

func (m *MemoryStore) DeleteTournament(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tournaments[id]; !ok {
		return fmt.Errorf("tournament %s not found", id)
	}

	delete(m.tournaments, id)
	return nil
}

func (m *MemoryStore) UpdateMatchResult(_ context.Context, tournamentID string, roundNumber int, matchID string, result models.MatchResult, score string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tournaments[tournamentID]
	if !ok {
		return fmt.Errorf("tournament %s not found", tournamentID)
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
				return nil
			}
		}
		return fmt.Errorf("match %s not found in round %d", matchID, roundNumber)
	}

	return fmt.Errorf("round %d not found", roundNumber)
}

func (m *MemoryStore) SetRoundPairings(_ context.Context, tournamentID string, roundNumber int, matches []models.Match) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tournaments[tournamentID]
	if !ok {
		return fmt.Errorf("tournament %s not found", tournamentID)
	}

	for i := range t.Rounds {
		if t.Rounds[i].Number == roundNumber {
			t.Rounds[i].Matches = matches
			t.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("round %d not found", roundNumber)
}

func (m *MemoryStore) UpdateHoleResult(_ context.Context, tournamentID string, roundNumber int, matchID string, hole int, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tournaments[tournamentID]
	if !ok {
		return fmt.Errorf("tournament %s not found", tournamentID)
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
				return nil
			}
		}
		return fmt.Errorf("match %s not found in round %d", matchID, roundNumber)
	}

	return fmt.Errorf("round %d not found", roundNumber)
}

func (m *MemoryStore) RegisterUser(_ context.Context, user *models.RegisteredUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.Email] = user
	return nil
}

func (m *MemoryStore) ListRegisteredUsers(_ context.Context) ([]*models.RegisteredUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.RegisteredUser, 0, len(m.users))
	for _, u := range m.users {
		copied := *u
		result = append(result, &copied)
	}
	return result, nil
}

func (m *MemoryStore) LinkPlayer(_ context.Context, tournamentID string, playerID string, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tournaments[tournamentID]
	if !ok {
		return fmt.Errorf("tournament %s not found", tournamentID)
	}

	for ti := range t.Teams {
		for pi := range t.Teams[ti].Players {
			if t.Teams[ti].Players[pi].ID == playerID {
				t.Teams[ti].Players[pi].UserEmail = email
				t.UpdatedAt = time.Now()
				return nil
			}
		}
	}

	return fmt.Errorf("player %s not found", playerID)
}
