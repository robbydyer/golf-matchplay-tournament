package models

import "time"

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
	ID     string `json:"id"`
	Name   string `json:"name"`
	TeamID string `json:"teamId"`
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
