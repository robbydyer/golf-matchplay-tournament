package store

import (
	"context"
	"fmt"
	"scoring-backend/internal/models"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FirestoreStore struct {
	client    *firestore.Client
	projectID string
}

func NewFirestoreStore(ctx context.Context, projectID, databaseID string) (*FirestoreStore, error) {
	if databaseID == "" {
		databaseID = firestore.DefaultDatabaseID
	}
	client, err := firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	if err != nil {
		return nil, fmt.Errorf("creating firestore client: %w", err)
	}
	return &FirestoreStore{client: client, projectID: projectID}, nil
}

func (f *FirestoreStore) Close() error {
	return f.client.Close()
}

func (f *FirestoreStore) tournaments() *firestore.CollectionRef {
	return f.client.Collection("tournaments")
}

func (f *FirestoreStore) registeredUsers() *firestore.CollectionRef {
	return f.client.Collection("registered_users")
}

func (f *FirestoreStore) localUsers() *firestore.CollectionRef {
	return f.client.Collection("local_users")
}

// normalizeTournament ensures all nil slices and maps are initialized after
// reading from Firestore, which does not preserve empty slices/maps.
func normalizeTournament(t *models.Tournament) {
	for i := range t.Teams {
		if t.Teams[i].Players == nil {
			t.Teams[i].Players = []models.Player{}
		}
	}
	if t.Rounds == nil {
		t.Rounds = []models.Round{}
	}
	for i := range t.Rounds {
		if t.Rounds[i].Matches == nil {
			t.Rounds[i].Matches = []models.Match{}
		}
		for j := range t.Rounds[i].Matches {
			if t.Rounds[i].Matches[j].HoleResults == nil {
				t.Rounds[i].Matches[j].HoleResults = make(map[string]string)
			}
			if t.Rounds[i].Matches[j].Team1Players == nil {
				t.Rounds[i].Matches[j].Team1Players = []string{}
			}
			if t.Rounds[i].Matches[j].Team2Players == nil {
				t.Rounds[i].Matches[j].Team2Players = []string{}
			}
		}
	}
}

// --- Tournament CRUD ---

func (f *FirestoreStore) CreateTournament(ctx context.Context, t *models.Tournament) error {
	ref := f.tournaments().Doc(t.ID)

	// Check if already exists
	_, err := ref.Get(ctx)
	if err == nil {
		return fmt.Errorf("tournament %s already exists", t.ID)
	}
	if status.Code(err) != codes.NotFound {
		return fmt.Errorf("checking tournament %s: %w", t.ID, err)
	}

	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	if _, err := ref.Set(ctx, t); err != nil {
		return fmt.Errorf("creating tournament %s: %w", t.ID, err)
	}
	return nil
}

// ImportTournament writes a tournament preserving its original timestamps.
// Used for data migration. Overwrites any existing document with the same ID.
func (f *FirestoreStore) ImportTournament(ctx context.Context, t *models.Tournament) error {
	ref := f.tournaments().Doc(t.ID)
	if _, err := ref.Set(ctx, t); err != nil {
		return fmt.Errorf("importing tournament %s: %w", t.ID, err)
	}
	return nil
}

func (f *FirestoreStore) GetTournament(ctx context.Context, id string) (*models.Tournament, error) {
	doc, err := f.tournaments().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("tournament %s not found", id)
		}
		return nil, fmt.Errorf("getting tournament %s: %w", id, err)
	}

	var t models.Tournament
	if err := doc.DataTo(&t); err != nil {
		return nil, fmt.Errorf("decoding tournament %s: %w", id, err)
	}

	normalizeTournament(&t)
	return &t, nil
}

func (f *FirestoreStore) UpdateTournament(ctx context.Context, t *models.Tournament) error {
	ref := f.tournaments().Doc(t.ID)

	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("tournament %s not found", t.ID)
		}
		return fmt.Errorf("checking tournament %s: %w", t.ID, err)
	}

	t.UpdatedAt = time.Now()
	if _, err := ref.Set(ctx, t); err != nil {
		return fmt.Errorf("updating tournament %s: %w", t.ID, err)
	}
	return nil
}

func (f *FirestoreStore) ListTournaments(ctx context.Context) ([]*models.Tournament, error) {
	iter := f.tournaments().Documents(ctx)
	defer iter.Stop()

	tournaments := make([]*models.Tournament, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing tournaments: %w", err)
		}

		var t models.Tournament
		if err := doc.DataTo(&t); err != nil {
			continue // skip corrupt documents
		}
		normalizeTournament(&t)
		tournaments = append(tournaments, &t)
	}
	return tournaments, nil
}

func (f *FirestoreStore) DeleteTournament(ctx context.Context, id string) error {
	ref := f.tournaments().Doc(id)

	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("tournament %s not found", id)
		}
		return fmt.Errorf("checking tournament %s: %w", id, err)
	}

	if _, err := ref.Delete(ctx); err != nil {
		return fmt.Errorf("deleting tournament %s: %w", id, err)
	}
	return nil
}

// --- Match operations ---

// getTournamentForUpdate reads a tournament and returns it along with its doc ref.
func (f *FirestoreStore) getTournamentForUpdate(ctx context.Context, tournamentID string) (*models.Tournament, *firestore.DocumentRef, error) {
	ref := f.tournaments().Doc(tournamentID)
	doc, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil, fmt.Errorf("tournament %s not found", tournamentID)
		}
		return nil, nil, fmt.Errorf("getting tournament %s: %w", tournamentID, err)
	}

	var t models.Tournament
	if err := doc.DataTo(&t); err != nil {
		return nil, nil, fmt.Errorf("decoding tournament %s: %w", tournamentID, err)
	}
	normalizeTournament(&t)

	return &t, ref, nil
}

func (f *FirestoreStore) UpdateMatchResult(ctx context.Context, tournamentID string, roundNumber int, matchID string, result models.MatchResult, score string) error {
	t, ref, err := f.getTournamentForUpdate(ctx, tournamentID)
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
				if _, err := ref.Set(ctx, t); err != nil {
					return fmt.Errorf("updating tournament %s: %w", tournamentID, err)
				}
				return nil
			}
		}
		return fmt.Errorf("match %s not found in round %d", matchID, roundNumber)
	}

	return fmt.Errorf("round %d not found", roundNumber)
}

func (f *FirestoreStore) SetRoundPairings(ctx context.Context, tournamentID string, roundNumber int, matches []models.Match) error {
	t, ref, err := f.getTournamentForUpdate(ctx, tournamentID)
	if err != nil {
		return err
	}

	for i := range t.Rounds {
		if t.Rounds[i].Number == roundNumber {
			t.Rounds[i].Matches = matches
			t.UpdatedAt = time.Now()
			if _, err := ref.Set(ctx, t); err != nil {
				return fmt.Errorf("updating tournament %s: %w", tournamentID, err)
			}
			return nil
		}
	}

	return fmt.Errorf("round %d not found", roundNumber)
}

func (f *FirestoreStore) UpdateHoleResult(ctx context.Context, tournamentID string, roundNumber int, matchID string, hole int, result string) error {
	t, ref, err := f.getTournamentForUpdate(ctx, tournamentID)
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
					match.HoleResults = make(map[string]string)
				}
				key := strconv.Itoa(hole)
				if result == "" {
					delete(match.HoleResults, key)
				} else {
					match.HoleResults[key] = result
				}
				// Backfill earlier empty holes as halved
				for h := 1; h < hole; h++ {
					k := strconv.Itoa(h)
					if match.HoleResults[k] == "" {
						match.HoleResults[k] = "halved"
					}
				}
				match.Result, match.Score = models.CalculateMatchPlayResult(match.HoleResults, t.Teams[0].Name, t.Teams[1].Name)
				t.UpdatedAt = time.Now()
				if _, err := ref.Set(ctx, t); err != nil {
					return fmt.Errorf("updating tournament %s: %w", tournamentID, err)
				}
				return nil
			}
		}
		return fmt.Errorf("match %s not found in round %d", matchID, roundNumber)
	}

	return fmt.Errorf("round %d not found", roundNumber)
}

// --- User registry ---

func (f *FirestoreStore) RegisterUser(ctx context.Context, user *models.RegisteredUser) error {
	ref := f.registeredUsers().Doc(user.Email)
	if _, err := ref.Set(ctx, user); err != nil {
		return fmt.Errorf("registering user: %w", err)
	}
	return nil
}

func (f *FirestoreStore) ListRegisteredUsers(ctx context.Context) ([]*models.RegisteredUser, error) {
	iter := f.registeredUsers().Documents(ctx)
	defer iter.Stop()

	result := make([]*models.RegisteredUser, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing registered users: %w", err)
		}

		var u models.RegisteredUser
		if err := doc.DataTo(&u); err != nil {
			continue
		}
		result = append(result, &u)
	}
	return result, nil
}

// --- Player-user linking ---

func (f *FirestoreStore) LinkPlayer(ctx context.Context, tournamentID string, playerID string, email string) error {
	t, ref, err := f.getTournamentForUpdate(ctx, tournamentID)
	if err != nil {
		return err
	}

	for ti := range t.Teams {
		for pi := range t.Teams[ti].Players {
			if t.Teams[ti].Players[pi].ID == playerID {
				t.Teams[ti].Players[pi].UserEmail = email
				t.UpdatedAt = time.Now()
				if _, err := ref.Set(ctx, t); err != nil {
					return fmt.Errorf("updating tournament %s: %w", tournamentID, err)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("player %s not found", playerID)
}

// --- Local user registration ---

func (f *FirestoreStore) CreateLocalUser(ctx context.Context, user *models.LocalUser) error {
	key := strings.ToLower(user.Email)
	ref := f.localUsers().Doc(key)

	_, err := ref.Get(ctx)
	if err == nil {
		return fmt.Errorf("a user with email %s already exists", user.Email)
	}
	if status.Code(err) != codes.NotFound {
		return fmt.Errorf("checking user %s: %w", user.Email, err)
	}

	if _, err := ref.Set(ctx, user); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

func (f *FirestoreStore) GetLocalUser(ctx context.Context, email string) (*models.LocalUser, error) {
	key := strings.ToLower(email)
	doc, err := f.localUsers().Doc(key).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("getting user: %w", err)
	}

	var u models.LocalUser
	if err := doc.DataTo(&u); err != nil {
		return nil, fmt.Errorf("decoding user: %w", err)
	}
	return &u, nil
}

func (f *FirestoreStore) VerifyLocalUser(ctx context.Context, token string) error {
	iter := f.localUsers().Where("VerificationToken", "==", token).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("invalid verification token")
	}
	if err != nil {
		return fmt.Errorf("querying verification token: %w", err)
	}

	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "EmailVerified", Value: true},
		{Path: "VerificationToken", Value: ""},
	})
	if err != nil {
		return fmt.Errorf("updating user verification: %w", err)
	}
	return nil
}

func (f *FirestoreStore) ListLocalUsers(ctx context.Context) ([]*models.LocalUser, error) {
	iter := f.localUsers().Documents(ctx)
	defer iter.Stop()

	result := make([]*models.LocalUser, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing local users: %w", err)
		}

		var u models.LocalUser
		if err := doc.DataTo(&u); err != nil {
			continue
		}
		result = append(result, &u)
	}
	return result, nil
}

func (f *FirestoreStore) ConfirmLocalUser(ctx context.Context, email string) error {
	key := strings.ToLower(email)
	ref := f.localUsers().Doc(key)

	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("getting user: %w", err)
	}

	_, err = ref.Update(ctx, []firestore.Update{
		{Path: "Confirmed", Value: true},
	})
	if err != nil {
		return fmt.Errorf("confirming user: %w", err)
	}
	return nil
}

func (f *FirestoreStore) DeleteLocalUser(ctx context.Context, email string) error {
	key := strings.ToLower(email)
	ref := f.localUsers().Doc(key)

	_, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("getting user: %w", err)
	}

	if _, err := ref.Delete(ctx); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}
