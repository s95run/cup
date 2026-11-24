package s95client

import (
	"context"
	"strings"
	"time"
)

// Mock is a local stand-in for s95.ru used when s95.mock is enabled.
type Mock struct {
	Events   []Event
	Athletes []AthleteSearchHit
	Protocol *ProtocolResponse
}

func NewMock() *Mock {
	return &Mock{
		Events: []Event{
			{CodeName: "angarskieprudy", Name: "Ангарские пруды", Town: "Москва", Active: true},
			{CodeName: "pechatniki", Name: "Пристань Печатники", Town: "Москва", Active: true},
		},
		Athletes: []AthleteSearchHit{
			{ID: 101, Code: 770000101, Name: "Анна ИВАНОВА", Gender: "female", HomeEventCode: "angarskieprudy", HomeEventName: "Ангарские пруды", ClubName: "Wake&Run", RunsCount: 12, VolunteeringByRole: map[string]int{"marshal": 1}},
			{ID: 102, Code: 770000102, Name: "Мария ПЕТРОВА", Gender: "female", HomeEventCode: "angarskieprudy", HomeEventName: "Ангарские пруды", ClubName: "Wake&Run", RunsCount: 0, VolunteeringByRole: map[string]int{"marshal": 4, "other": 2}},
			{ID: 103, Code: 770000103, Name: "Иван СИДОРОВ", Gender: "male", HomeEventCode: "pechatniki", HomeEventName: "Пристань Печатники", ClubName: "", RunsCount: 4},
			{ID: 104, Code: 770000104, Name: "Пётр ОРЛОВ", Gender: "male", HomeEventCode: "angarskieprudy", HomeEventName: "Ангарские пруды", ClubName: "", RunsCount: 2},
			{ID: 105, Code: 770000105, Name: "Олег НОВИКОВ", Gender: "male", HomeEventCode: "angarskieprudy", HomeEventName: "Ангарские пруды", ClubName: "", RunsCount: 8, VolunteeringByRole: map[string]int{"timer": 2}},
		},
	}
}

func (m *Mock) ListEvents(context.Context) ([]Event, error) {
	return append([]Event(nil), m.Events...), nil
}

func (m *Mock) SearchAthletes(_ context.Context, q string) ([]AthleteSearchHit, error) {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []AthleteSearchHit
	for _, athlete := range m.Athletes {
		if q == "" || strings.Contains(strings.ToLower(athlete.Name), q) || strings.Contains(strings.ToLower(athlete.HomeEventCode), q) {
			athlete.ApplyQualifyingVols()
			out = append(out, athlete)
		}
	}
	if len(out) > 20 {
		out = out[:20]
	}
	return out, nil
}

func (m *Mock) FetchProtocol(_ context.Context, _, _ string, knownUpdatedAt *time.Time) (*ProtocolResponse, error) {
	if m.Protocol != nil {
		return m.Protocol, nil
	}
	results := make([]ProtocolResult, 0, len(m.Athletes))
	for i, athlete := range m.Athletes {
		results = append(results, ProtocolResult{AthleteID: athlete.ID, Position: i + 1, TotalTimeSec: 1400 + i*30})
	}
	return &ProtocolResponse{
		Published:  true,
		ActivityID: 9001,
		UpdatedAt:  time.Now().UTC(),
		Results:    results,
	}, nil
}
