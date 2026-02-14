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

	// User registry
	RegisterUser(ctx context.Context, user *models.RegisteredUser) error
	ListRegisteredUsers(ctx context.Context) ([]*models.RegisteredUser, error)

	// Player-user linking
	LinkPlayer(ctx context.Context, tournamentID string, playerID string, email string) error

	// Local user registration
	CreateLocalUser(ctx context.Context, user *models.LocalUser) error
	GetLocalUser(ctx context.Context, email string) (*models.LocalUser, error)
	VerifyLocalUser(ctx context.Context, token string) error
	ListLocalUsers(ctx context.Context) ([]*models.LocalUser, error)
	ConfirmLocalUser(ctx context.Context, email string) error
	DeleteLocalUser(ctx context.Context, email string) error
}
