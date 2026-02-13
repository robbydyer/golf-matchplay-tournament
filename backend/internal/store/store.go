package store

import (
	"context"
	"scoring-backend/internal/models"
)

// Store defines the interface for tournament data persistence.
// Implementations can back this with in-memory storage, Firestore, or any other provider.
type Store interface {
	// Tournament CRUD
	CreateTournament(ctx context.Context, t *models.Tournament) error
	GetTournament(ctx context.Context, id string) (*models.Tournament, error)
	UpdateTournament(ctx context.Context, t *models.Tournament) error
	ListTournaments(ctx context.Context) ([]*models.Tournament, error)
	DeleteTournament(ctx context.Context, id string) error

	// Match operations
	UpdateMatchResult(ctx context.Context, tournamentID string, roundNumber int, matchID string, result models.MatchResult, score string) error
	SetRoundPairings(ctx context.Context, tournamentID string, roundNumber int, matches []models.Match) error
	UpdateHoleResult(ctx context.Context, tournamentID string, roundNumber int, matchID string, hole int, result string) error
}
