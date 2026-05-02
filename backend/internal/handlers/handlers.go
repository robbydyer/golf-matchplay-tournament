package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"scoring-backend/internal/auth"
	"scoring-backend/internal/email"
	"scoring-backend/internal/models"
	"scoring-backend/internal/store"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	store       store.Store
	emailCfg    *email.Config
	jwtSecret   string
	appURL      string
	adminEmails map[string]bool
}

func New(s store.Store, emailCfg *email.Config, jwtSecret, appURL string, adminEmails map[string]bool) *Handler {
	return &Handler{
		store:       s,
		emailCfg:    emailCfg,
		jwtSecret:   jwtSecret,
		appURL:      appURL,
		adminEmails: adminEmails,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Public auth routes (no auth middleware)
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/verify", h.VerifyEmail)

	// Authenticated routes
	mux.HandleFunc("GET /api/me", h.GetMe)
	mux.HandleFunc("GET /api/tournaments", h.ListTournaments)
	mux.HandleFunc("POST /api/tournaments", auth.RequireAdmin(h.CreateTournament))
	mux.HandleFunc("GET /api/tournaments/{id}", h.GetTournament)
	mux.HandleFunc("PUT /api/tournaments/{id}", auth.RequireAdmin(h.UpdateTournament))
	mux.HandleFunc("DELETE /api/tournaments/{id}", auth.RequireAdmin(h.DeleteTournament))
	mux.HandleFunc("GET /api/tournaments/{id}/scoreboard", h.GetScoreboard)
	mux.HandleFunc("PUT /api/tournaments/{id}/lock", auth.RequireAdmin(h.LockTournament))
	mux.HandleFunc("PUT /api/tournaments/{id}/combine-rounds", auth.RequireAdmin(h.CombineRounds))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/name", auth.RequireAdmin(h.UpdateRoundName))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/holes", auth.RequireAdmin(h.UpdateRoundHoles))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/points", auth.RequireAdmin(h.UpdateRoundPoints))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/lock", auth.RequireAdmin(h.LockRound))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/pairings", auth.RequireAdmin(h.SetPairings))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/matches/{matchId}", auth.RequireAdmin(h.UpdateMatchResult))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/matches/{matchId}/holes/{hole}", h.UpdateHoleResult)
	mux.HandleFunc("GET /api/tournaments/{id}/rankings", h.GetRankings)
	mux.HandleFunc("PUT /api/tournaments/{id}/rankings", h.SubmitRanking)
	mux.HandleFunc("PUT /api/tournaments/{id}/rankings/lock", auth.RequireAdmin(h.LockRankings))
	mux.HandleFunc("GET /api/users", auth.RequireAdmin(h.ListUsers))
	mux.HandleFunc("PUT /api/tournaments/{id}/players/{playerId}/link", auth.RequireAdmin(h.LinkPlayer))

	// Admin user management
	mux.HandleFunc("GET /api/admin/users", auth.RequireAdmin(h.ListLocalUsersAdmin))
	mux.HandleFunc("POST /api/admin/users/confirm", auth.RequireAdmin(h.ConfirmUser))
	mux.HandleFunc("POST /api/admin/users/reject", auth.RequireAdmin(h.RejectUser))
	mux.HandleFunc("POST /api/admin/users/enable", auth.RequireAdmin(h.EnableUser))
}

// --- Public auth handlers ---

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Name == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email, name, and password are required")
		return
	}

	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	verToken, err := auth.GenerateVerificationToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate verification token")
		return
	}

	user := &models.LocalUser{
		Email:             req.Email,
		Name:              req.Name,
		PasswordHash:      string(hash),
		EmailVerified:     false,
		Confirmed:         h.adminEmails[req.Email],
		VerificationToken: verToken,
		CreatedAt:         time.Now(),
	}

	if err := h.store.CreateLocalUser(r.Context(), user); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	if h.emailCfg.IsConfigured() {
		if err := h.emailCfg.SendVerification(req.Email, verToken, h.appURL); err != nil {
			log.Printf("Failed to send verification email to %s: %v", req.Email, err)
		}
		// Notify admins about the new registration
		if !user.Confirmed {
			admins := make([]string, 0, len(h.adminEmails))
			for em := range h.adminEmails {
				admins = append(admins, em)
			}
			if err := h.emailCfg.SendNewUserNotification(admins, req.Name, req.Email, h.appURL); err != nil {
				log.Printf("Failed to send admin notification: %v", err)
			}
		}
	} else {
		log.Printf("Email not configured. Verification token for %s: %s", req.Email, verToken)
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "Registration successful. Please check your email to verify your account. An admin will need to confirm your access.",
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	user, err := h.store.GetLocalUser(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !user.EmailVerified {
		writeError(w, http.StatusForbidden, "please verify your email before logging in")
		return
	}

	if !user.Confirmed {
		writeError(w, http.StatusForbidden, "your account is pending admin approval")
		return
	}

	if user.Disabled {
		writeError(w, http.StatusForbidden, "your account has been disabled")
		return
	}

	token, err := auth.GenerateLocalToken(user.Email, user.Name, h.jwtSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
	})
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	if err := h.store.VerifyLocalUser(r.Context(), req.Token); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully. An admin will review your account before you can log in.",
	})
}

// --- Admin user management handlers ---

func (h *Handler) ListLocalUsersAdmin(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListLocalUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type userResponse struct {
		Email         string    `json:"email"`
		Name          string    `json:"name"`
		EmailVerified bool      `json:"emailVerified"`
		Confirmed     bool      `json:"confirmed"`
		Disabled      bool      `json:"disabled"`
		CreatedAt     time.Time `json:"createdAt"`
	}

	result := make([]userResponse, len(users))
	for i, u := range users {
		result[i] = userResponse{
			Email:         u.Email,
			Name:          u.Name,
			EmailVerified: u.EmailVerified,
			Confirmed:     u.Confirmed,
			Disabled:      u.Disabled,
			CreatedAt:     u.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ConfirmUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	if err := h.store.ConfirmLocalUser(r.Context(), req.Email); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user confirmed"})
}

func (h *Handler) RejectUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	if err := h.store.DeleteLocalUser(r.Context(), req.Email); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

func (h *Handler) EnableUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	if err := h.store.EnableLocalUser(r.Context(), req.Email); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user enabled"})
}

// --- Authenticated handlers ---

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	// Register/update user in the registry
	h.store.RegisterUser(r.Context(), &models.RegisteredUser{
		Email:   user.Email,
		Name:    user.Name,
		Picture: user.Picture,
	})

	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) ListTournaments(w http.ResponseWriter, r *http.Request) {
	tournaments, err := h.store.ListTournaments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tournaments)
}

type CreateTournamentRequest struct {
	Name      string `json:"name"`
	Team1Name string `json:"team1Name"`
	Team2Name string `json:"team2Name"`
}

func (h *Handler) CreateTournament(w http.ResponseWriter, r *http.Request) {
	var req CreateTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Team1Name == "" || req.Team2Name == "" {
		writeError(w, http.StatusBadRequest, "name, team1Name, and team2Name are required")
		return
	}

	t := &models.Tournament{
		ID:   uuid.New().String(),
		Name: req.Name,
		Teams: [2]models.Team{
			{ID: uuid.New().String(), Name: req.Team1Name, Players: []models.Player{}},
			{ID: uuid.New().String(), Name: req.Team2Name, Players: []models.Player{}},
		},
		Rounds: models.DefaultRounds(),
	}

	if err := h.store.CreateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) GetTournament(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type UpdateTournamentRequest struct {
	Name        string        `json:"name,omitempty"`
	Teams       *[2]TeamInput `json:"teams,omitempty"`
	HeaderColor string        `json:"headerColor,omitempty"`
	BgColor     string        `json:"bgColor,omitempty"`
}

type TeamInput struct {
	Name    string        `json:"name"`
	Color   string        `json:"color,omitempty"`
	Logo    string        `json:"logo,omitempty"`
	Players []PlayerInput `json:"players"`
}

type PlayerInput struct {
	Name string `json:"name"`
}

func (h *Handler) UpdateTournament(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req UpdateTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		t.Name = req.Name
	}

	if req.HeaderColor != "" {
		t.HeaderColor = req.HeaderColor
	}
	if req.BgColor != "" {
		t.BgColor = req.BgColor
	}

	if req.Teams != nil {
		for i := 0; i < 2; i++ {
			t.Teams[i].Name = req.Teams[i].Name
			if req.Teams[i].Color != "" {
				t.Teams[i].Color = req.Teams[i].Color
			}
			t.Teams[i].Logo = req.Teams[i].Logo
			players := make([]models.Player, len(req.Teams[i].Players))
			for j, p := range req.Teams[i].Players {
				playerID := uuid.New().String()
				userEmail := ""
				if j < len(t.Teams[i].Players) {
					playerID = t.Teams[i].Players[j].ID
					userEmail = t.Teams[i].Players[j].UserEmail
				}
				players[j] = models.Player{
					ID:        playerID,
					Name:      p.Name,
					TeamID:    t.Teams[i].ID,
					UserEmail: userEmail,
				}
			}
			t.Teams[i].Players = players
		}
	}

	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) DeleteTournament(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteTournament(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetScoreboard(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	scoreboard := t.CalculateScoreboard()
	writeJSON(w, http.StatusOK, scoreboard)
}

type SetPairingsRequest struct {
	Matches []MatchInput `json:"matches"`
}

type MatchInput struct {
	Team1Players []string `json:"team1Players"`
	Team2Players []string `json:"team2Players"`
}

func (h *Handler) LockTournament(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		Locked bool `json:"locked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t.Locked = req.Locked
	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) CombineRounds(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		Combine bool `json:"combine"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t.CombineRounds23 = req.Combine
	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) LockRound(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	roundNum, err := strconv.Atoi(r.PathValue("round"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	var req struct {
		Locked bool `json:"locked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	found := false
	for i := range t.Rounds {
		if t.Rounds[i].Number == roundNum {
			t.Rounds[i].Locked = req.Locked
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, fmt.Sprintf("round %d not found", roundNum))
		return
	}

	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) UpdateRoundPoints(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	roundNum, err := strconv.Atoi(r.PathValue("round"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	var req struct {
		PointsPerMatch float64 `json:"pointsPerMatch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PointsPerMatch <= 0 {
		writeError(w, http.StatusBadRequest, "pointsPerMatch must be greater than 0")
		return
	}

	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	found := false
	for i := range t.Rounds {
		if t.Rounds[i].Number == roundNum {
			t.Rounds[i].PointsPerMatch = req.PointsPerMatch
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, fmt.Sprintf("round %d not found", roundNum))
		return
	}

	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) UpdateRoundHoles(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	roundNum, err := strconv.Atoi(r.PathValue("round"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	var req struct {
		Holes int `json:"holes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Holes < 1 || req.Holes > 18 {
		writeError(w, http.StatusBadRequest, "holes must be between 1 and 18")
		return
	}

	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	found := false
	for i := range t.Rounds {
		if t.Rounds[i].Number == roundNum {
			t.Rounds[i].Holes = req.Holes
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, fmt.Sprintf("round %d not found", roundNum))
		return
	}

	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) UpdateRoundName(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	roundNum, err := strconv.Atoi(r.PathValue("round"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	found := false
	for i := range t.Rounds {
		if t.Rounds[i].Number == roundNum {
			t.Rounds[i].Name = req.Name
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, fmt.Sprintf("round %d not found", roundNum))
		return
	}

	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) SetPairings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	roundStr := r.PathValue("round")
	roundNum, err := strconv.Atoi(roundStr)
	if err != nil || roundNum < 1 || roundNum > 5 {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	var req SetPairingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	matches := make([]models.Match, len(req.Matches))
	for i, m := range req.Matches {
		matches[i] = models.Match{
			ID:           uuid.New().String(),
			RoundNumber:  roundNum,
			Team1Players: m.Team1Players,
			Team2Players: m.Team2Players,
			Result:       models.ResultPending,
			HoleResults:  make(map[string]string),
		}
	}

	if err := h.store.SetRoundPairings(r.Context(), id, roundNum, matches); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	t, _ := h.store.GetTournament(r.Context(), id)
	writeJSON(w, http.StatusOK, t)
}

type UpdateMatchResultRequest struct {
	Result models.MatchResult `json:"result"`
	Score  string             `json:"score"`
}

func (h *Handler) UpdateMatchResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	roundStr := r.PathValue("round")
	matchID := r.PathValue("matchId")

	roundNum, err := strconv.Atoi(roundStr)
	if err != nil || roundNum < 1 || roundNum > 5 {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	var req UpdateMatchResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validResults := map[models.MatchResult]bool{
		models.ResultPending: true,
		models.ResultTeam1:   true,
		models.ResultTeam2:   true,
		models.ResultTie:     true,
	}
	if !validResults[req.Result] {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid result: %s", req.Result))
		return
	}

	if err := h.store.UpdateMatchResult(r.Context(), id, roundNum, matchID, req.Result, req.Score); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	t, _ := h.store.GetTournament(r.Context(), id)
	writeJSON(w, http.StatusOK, t)
}

type UpdateHoleResultRequest struct {
	Result string `json:"result"`
}

func (h *Handler) UpdateHoleResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	roundStr := r.PathValue("round")
	matchID := r.PathValue("matchId")
	holeStr := r.PathValue("hole")

	roundNum, err := strconv.Atoi(roundStr)
	if err != nil || roundNum < 1 || roundNum > 5 {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	holeNum, err := strconv.Atoi(holeStr)
	if err != nil || holeNum < 1 || holeNum > 18 {
		writeError(w, http.StatusBadRequest, "invalid hole number (1-18)")
		return
	}

	// Validate hole number against round's configured hole count
	{
		t, err := h.store.GetTournament(r.Context(), id)
		if err == nil {
			for _, round := range t.Rounds {
				if round.Number == roundNum && holeNum > round.HoleCount() {
					writeError(w, http.StatusBadRequest, fmt.Sprintf("hole %d exceeds this round's %d holes", holeNum, round.HoleCount()))
					return
				}
			}
		}
	}

	var req UpdateHoleResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validResults := map[string]bool{"team1": true, "team2": true, "halved": true, "": true}
	if !validResults[req.Result] {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid hole result: %s", req.Result))
		return
	}

	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	if !user.IsAdmin {
		t, err := h.store.GetTournament(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if t.Locked {
			writeError(w, http.StatusForbidden, "this tournament is locked")
			return
		}
		for _, round := range t.Rounds {
			if round.Number == roundNum && round.Locked {
				writeError(w, http.StatusForbidden, "this round is locked")
				return
			}
		}
		if !isPlayerInMatch(t, roundNum, matchID, strings.ToLower(user.Email)) {
			writeError(w, http.StatusForbidden, "you are not a player in this match")
			return
		}
	}

	if err := h.store.UpdateHoleResult(r.Context(), id, roundNum, matchID, holeNum, req.Result); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	t, _ := h.store.GetTournament(r.Context(), id)
	writeJSON(w, http.StatusOK, t)
}

func isPlayerInMatch(t *models.Tournament, roundNumber int, matchID string, email string) bool {
	playerEmails := make(map[string]string)
	for _, team := range t.Teams {
		for _, p := range team.Players {
			if p.UserEmail != "" {
				playerEmails[p.ID] = strings.ToLower(p.UserEmail)
			}
		}
	}

	for _, round := range t.Rounds {
		if round.Number != roundNumber {
			continue
		}
		for _, match := range round.Matches {
			if match.ID != matchID {
				continue
			}
			for _, pid := range match.Team1Players {
				if playerEmails[pid] == email {
					return true
				}
			}
			for _, pid := range match.Team2Players {
				if playerEmails[pid] == email {
					return true
				}
			}
			return false
		}
	}
	return false
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	localUsers, err := h.store.ListLocalUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	users := make([]*models.RegisteredUser, 0, len(localUsers))
	for _, u := range localUsers {
		if u.Confirmed {
			users = append(users, &models.RegisteredUser{
				Email: u.Email,
				Name:  u.Name,
			})
		}
	}
	writeJSON(w, http.StatusOK, users)
}

type LinkPlayerRequest struct {
	Email string `json:"email"`
}

func (h *Handler) LinkPlayer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	playerID := r.PathValue("playerId")

	var req LinkPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.store.LinkPlayer(r.Context(), id, playerID, req.Email); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	t, _ := h.store.GetTournament(r.Context(), id)
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) LockRankings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		Locked bool `json:"locked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t.RankingsLocked = req.Locked
	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) GetRankings(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	id := r.PathValue("id")
	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if !user.IsAdmin {
		// Non-admins only get their own ranking back
		for _, rk := range t.Rankings {
			if strings.EqualFold(rk.SubmittedBy, user.Email) {
				writeJSON(w, http.StatusOK, []models.PlayerRanking{rk})
				return
			}
		}
		writeJSON(w, http.StatusOK, []models.PlayerRanking{})
		return
	}

	if t.Rankings == nil {
		writeJSON(w, http.StatusOK, []models.PlayerRanking{})
		return
	}
	writeJSON(w, http.StatusOK, t.Rankings)
}

func (h *Handler) SubmitRanking(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	id := r.PathValue("id")
	t, err := h.store.GetTournament(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if t.RankingsLocked && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "rankings are locked")
		return
	}

	var req struct {
		PlayerIDs []string `json:"playerIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Determine which team the submitter belongs to
	submitterEmail := strings.ToLower(user.Email)
	var teamPlayers []models.Player
	for _, team := range t.Teams {
		for _, p := range team.Players {
			if strings.EqualFold(p.UserEmail, submitterEmail) {
				teamPlayers = team.Players
				break
			}
		}
		if teamPlayers != nil {
			break
		}
	}
	if teamPlayers == nil && !user.IsAdmin {
		writeError(w, http.StatusForbidden, "you are not linked to a player on this tournament")
		return
	}

	// Validate: submitted player IDs must exactly match the team's players
	if teamPlayers != nil {
		teamIDs := make(map[string]bool)
		for _, p := range teamPlayers {
			teamIDs[p.ID] = true
		}
		if len(req.PlayerIDs) != len(teamPlayers) {
			writeError(w, http.StatusBadRequest, "ranking must include all team members")
			return
		}
		seen := make(map[string]bool)
		for _, pid := range req.PlayerIDs {
			if !teamIDs[pid] {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("player %s is not on your team", pid))
				return
			}
			if seen[pid] {
				writeError(w, http.StatusBadRequest, "duplicate player in ranking")
				return
			}
			seen[pid] = true
		}
	}

	// Upsert the ranking
	found := false
	for i, rk := range t.Rankings {
		if strings.EqualFold(rk.SubmittedBy, submitterEmail) {
			t.Rankings[i].PlayerIDs = req.PlayerIDs
			t.Rankings[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}
	if !found {
		t.Rankings = append(t.Rankings, models.PlayerRanking{
			SubmittedBy: submitterEmail,
			PlayerIDs:   req.PlayerIDs,
			UpdatedAt:   time.Now(),
		})
	}

	if err := h.store.UpdateTournament(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "ranking saved"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
