package domain

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

type AthleteQualification struct {
	S95AthleteID int64
	Gender       Gender
	RunsCount    int
	VolCount     int
}

type RosterMember struct {
	S95AthleteID  int64
	Gender        Gender
	FinishTimeSec *int
	IsDNF         bool
}

type TeamStageInput struct {
	RegistrationID int64
	Members        []RosterMember
	BonusPoints    int
}

type StageTeamOutcome struct {
	RegistrationID     int64
	Disqualified       bool
	DSQReason          string
	TotalTimeSec       *int
	AvgTimeDS          *int
	FifthRunnerTimeSec *int
	Rank               *int
	BasePoints         int
	BonusPoints        int
	TotalPoints        int
}

const MinRuns = 1
const MinVolunteering = 4
const TeamSize = 5
const MinFemales = 2

type ScoringMode string

const (
	ScoringModeDynamicN ScoringMode = "dynamic_n"
	ScoringModeAvgTime  ScoringMode = "avg_time"
)
