package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"scoring-backend/internal/auth"
	"scoring-backend/internal/models"
	"scoring-backend/internal/store"
	"strconv"

	"github.com/google/uuid"
)

type Handler struct {
	store store.Store
}

func New(s store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/me", h.GetMe)
	mux.HandleFunc("GET /api/tournaments", h.ListTournaments)
	mux.HandleFunc("POST /api/tournaments", auth.RequireAdmin(h.CreateTournament))
	mux.HandleFunc("GET /api/tournaments/{id}", h.GetTournament)
	mux.HandleFunc("PUT /api/tournaments/{id}", auth.RequireAdmin(h.UpdateTournament))
	mux.HandleFunc("DELETE /api/tournaments/{id}", auth.RequireAdmin(h.DeleteTournament))
	mux.HandleFunc("GET /api/tournaments/{id}/scoreboard", h.GetScoreboard)
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/pairings", auth.RequireAdmin(h.SetPairings))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/matches/{matchId}", auth.RequireAdmin(h.UpdateMatchResult))
	mux.HandleFunc("PUT /api/tournaments/{id}/rounds/{round}/matches/{matchId}/holes/{hole}", h.UpdateHoleResult)
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
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
	Name  string        `json:"name,omitempty"`
	Teams *[2]TeamInput `json:"teams,omitempty"`
}

type TeamInput struct {
	Name    string              `json:"name"`
	Players []PlayerInput       `json:"players"`
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

	if req.Teams != nil {
		for i := 0; i < 2; i++ {
			t.Teams[i].Name = req.Teams[i].Name
			players := make([]models.Player, len(req.Teams[i].Players))
			for j, p := range req.Teams[i].Players {
				playerID := uuid.New().String()
				// Preserve existing player IDs if available
				if j < len(t.Teams[i].Players) {
					playerID = t.Teams[i].Players[j].ID
				}
				players[j] = models.Player{
					ID:     playerID,
					Name:   p.Name,
					TeamID: t.Teams[i].ID,
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
	Result string `json:"result"` // "team1", "team2", "halved", or ""
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

	if err := h.store.UpdateHoleResult(r.Context(), id, roundNum, matchID, holeNum-1, req.Result); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	t, _ := h.store.GetTournament(r.Context(), id)
	writeJSON(w, http.StatusOK, t)
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
