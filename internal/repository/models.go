package repository

import "time"

type Location struct {
	CodeName string `json:"code_name"`
	Name     string `json:"name"`
	Town     string `json:"town"`
	Timezone string `json:"timezone"`
	Active   bool   `json:"active"`
}

type User struct {
	ID           int64
	S95UserID    int64
	S95AthleteID int64
	Name         string
	Role         string
}

type Season struct {
	ID          int64  `json:"id"`
	Year        int    `json:"year"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	ScoringMode string `json:"scoring_mode"`
	MinStages   int    `json:"min_stages"`
	Published   bool   `json:"published"`
}

type Stage struct {
	ID                   int64      `json:"id"`
	SeasonID             int64      `json:"season_id"`
	StageNumber          int        `json:"stage_number"`
	EventCode            string     `json:"event_code"`
	EventName            string     `json:"event_name"`
	Timezone             string     `json:"timezone"`
	StageDate            time.Time  `json:"stage_date"`
	RegistrationClosesAt time.Time  `json:"registration_closes_at"`
	Status               string     `json:"status"`
	ActivityID           *int64     `json:"activity_id,omitempty"`
	ActivityUpdatedAt    *time.Time `json:"activity_updated_at,omitempty"`
}

type Team struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	HomeEventCode string `json:"home_event_code"`
	ClubName      string `json:"club_name"`
	CaptainUserID int64  `json:"captain_user_id"`
}

type MemberRow struct {
	RegistrationID   int64
	S95AthleteID     int64
	Name             string
	Gender           string
	HomeEvent        string
	Club             string
	QualRuns         int
	QualVols         int
	FinishTimeSec    *int
	ActivityPosition *int
	IsDNF            bool
}

type BonusRow struct {
	ID             int64
	RegistrationID int64
	Points         int
	Reason         string
}

type ResultRow struct {
	RegistrationID     int64  `json:"registration_id"`
	Disqualified       bool   `json:"disqualified"`
	DSQReason          string `json:"dsq_reason"`
	TotalTimeSec       *int   `json:"total_time_sec"`
	AvgTimeDS          *int   `json:"avg_time_ds"`
	FifthRunnerTimeSec *int   `json:"fifth_runner_time_sec"`
	Rank               *int   `json:"rank"`
	BasePoints         int    `json:"base_points"`
	BonusPoints        int    `json:"bonus_points"`
	TotalPoints        int    `json:"total_points"`
}

type RegistrationHead struct {
	ID            int64
	StageID       int64
	TeamID        int64
	TeamName      string
	TeamSlug      string
	HomeEventCode string
	ClubName      string
	CaptainUserID int64
}

type StandingRow struct {
	TeamID       int64
	Name         string
	Slug         string
	TotalPoints  int
	BasePoints   int
	BonusPoints  int
	StagesCount  int
	FirstPlaces  int
	AvgTimeTotal int
}
