package models

import (
	"fmt"
	"time"
)

type RoundType string

const (
	RoundLauderdale RoundType = "lauderdale"
	RoundFoursome   RoundType = "foursome"
	RoundFourBall   RoundType = "fourball"
	RoundSingles    RoundType = "singles"
)

type MatchResult string

const (
	ResultPending MatchResult = "pending"
	ResultTeam1   MatchResult = "team1"
	ResultTeam2   MatchResult = "team2"
	ResultTie     MatchResult = "tie"
)

type Player struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TeamID    string `json:"teamId"`
	UserEmail string `json:"userEmail,omitempty"`
}

type RegisteredUser struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type Team struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Players []Player `json:"players"`
}

type Match struct {
	ID           string      `json:"id"`
	RoundNumber  int         `json:"roundNumber"`
	Team1Players []string    `json:"team1Players"` // player IDs
	Team2Players []string    `json:"team2Players"` // player IDs
	Result       MatchResult `json:"result"`
	Score        string      `json:"score"`        // match play score, e.g. "2 & 1", "1 UP", "A/S"
	HoleResults  []string    `json:"holeResults"`   // per-hole results: "team1", "team2", "halved", or ""
}

type Round struct {
	Number         int       `json:"number"`
	Name           string    `json:"name"`
	Type           RoundType `json:"type"`
	PointsPerMatch float64   `json:"pointsPerMatch"`
	Matches        []Match   `json:"matches"`
}

type Tournament struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Teams     [2]Team   `json:"teams"`
	Rounds    []Round   `json:"rounds"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Scoreboard struct {
	Team1Name   string       `json:"team1Name"`
	Team2Name   string       `json:"team2Name"`
	Team1Total  float64      `json:"team1Total"`
	Team2Total  float64      `json:"team2Total"`
	RoundScores []RoundScore `json:"roundScores"`
}

type RoundScore struct {
	RoundNumber    int     `json:"roundNumber"`
	RoundName      string  `json:"roundName"`
	Team1Points    float64 `json:"team1Points"`
	Team2Points    float64 `json:"team2Points"`
	PointsPerMatch float64 `json:"pointsPerMatch"`
	MatchesPlayed  int     `json:"matchesPlayed"`
	TotalMatches   int     `json:"totalMatches"`
}

func DefaultRounds() []Round {
	return []Round{
		{
			Number:         1,
			Name:           "Lauderdale",
			Type:           RoundLauderdale,
			PointsPerMatch: 1.0,
			Matches:        []Match{},
		},
		{
			Number:         2,
			Name:           "Foursome (Alternate Shot) - Friday PM",
			Type:           RoundFoursome,
			PointsPerMatch: 0.5,
			Matches:        []Match{},
		},
		{
			Number:         3,
			Name:           "Foursome (Alternate Shot) - Saturday AM",
			Type:           RoundFoursome,
			PointsPerMatch: 0.5,
			Matches:        []Match{},
		},
		{
			Number:         4,
			Name:           "Four-Ball",
			Type:           RoundFourBall,
			PointsPerMatch: 1.0,
			Matches:        []Match{},
		},
		{
			Number:         5,
			Name:           "Singles",
			Type:           RoundSingles,
			PointsPerMatch: 1.0,
			Matches:        []Match{},
		},
	}
}

// CalculateMatchPlayResult derives the match result and score string from
// hole-by-hole results using standard match play rules. A match is clinched
// when a team leads by more holes than remain to be played.
func CalculateMatchPlayResult(holeResults []string, team1Name, team2Name string) (MatchResult, string) {
	if len(holeResults) == 0 {
		return ResultPending, ""
	}

	t1Wins := 0
	t2Wins := 0
	played := 0

	for _, r := range holeResults {
		switch r {
		case "team1":
			t1Wins++
			played++
		case "team2":
			t2Wins++
			played++
		case "halved":
			played++
		}
	}

	if played == 0 {
		return ResultPending, ""
	}

	lead := t1Wins - t2Wins
	remaining := 18 - played

	// Team 1 clinches
	if lead > 0 && lead > remaining {
		if remaining == 0 {
			return ResultTeam1, fmt.Sprintf("%d UP", lead)
		}
		return ResultTeam1, fmt.Sprintf("%d & %d", lead, remaining)
	}

	// Team 2 clinches
	if lead < 0 && -lead > remaining {
		if remaining == 0 {
			return ResultTeam2, fmt.Sprintf("%d UP", -lead)
		}
		return ResultTeam2, fmt.Sprintf("%d & %d", -lead, remaining)
	}

	// All 18 holes played, dead even
	if remaining == 0 && lead == 0 {
		return ResultTie, "A/S"
	}

	// Match still in progress — show running score
	if lead > 0 {
		return ResultPending, fmt.Sprintf("%s %d UP thru %d", team1Name, lead, played)
	}
	if lead < 0 {
		return ResultPending, fmt.Sprintf("%s %d UP thru %d", team2Name, -lead, played)
	}
	return ResultPending, fmt.Sprintf("A/S thru %d", played)
}

func (t *Tournament) CalculateScoreboard() Scoreboard {
	sb := Scoreboard{
		Team1Name: t.Teams[0].Name,
		Team2Name: t.Teams[1].Name,
	}

	for _, round := range t.Rounds {
		rs := RoundScore{
			RoundNumber:    round.Number,
			RoundName:      round.Name,
			PointsPerMatch: round.PointsPerMatch,
			TotalMatches:   len(round.Matches),
		}

		for _, match := range round.Matches {
			switch match.Result {
			case ResultTeam1:
				rs.Team1Points += round.PointsPerMatch
				rs.MatchesPlayed++
			case ResultTeam2:
				rs.Team2Points += round.PointsPerMatch
				rs.MatchesPlayed++
			case ResultTie:
				rs.Team1Points += round.PointsPerMatch / 2
				rs.Team2Points += round.PointsPerMatch / 2
				rs.MatchesPlayed++
			}
		}

		sb.Team1Total += rs.Team1Points
		sb.Team2Total += rs.Team2Points
		sb.RoundScores = append(sb.RoundScores, rs)
	}

	return sb
}
