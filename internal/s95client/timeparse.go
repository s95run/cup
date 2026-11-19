package s95client

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseTimeString converts S95 display times (MM:SS or HH:MM:SS) to seconds.
// Returns ok=false for blank, "xx:xx", or invalid input.
func ParseTimeString(s string) (sec int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "xx:xx") {
		return 0, false
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	parsePart := func(p string, max int) (int, bool) {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > max {
			return 0, false
		}
		return n, true
	}
	var hours, minutes, seconds int
	switch len(parts) {
	case 2:
		m, ok := parsePart(parts[0], 59)
		if !ok {
			return 0, false
		}
		secPart, ok := parsePart(parts[1], 59)
		if !ok {
			return 0, false
		}
		minutes, seconds = m, secPart
	case 3:
		h, ok := parsePart(parts[0], 23)
		if !ok {
			return 0, false
		}
		m, ok := parsePart(parts[1], 59)
		if !ok {
			return 0, false
		}
		sPart, ok := parsePart(parts[2], 59)
		if !ok {
			return 0, false
		}
		hours, minutes, seconds = h, m, sPart
	}
	return hours*3600 + minutes*60 + seconds, true
}

func parseActivityResults(body activityProtocolJSON) ([]ProtocolResult, error) {
	out := make([]ProtocolResult, 0, len(body.Results))
	for _, row := range body.Results {
		if row.Athlete.ID == nil || *row.Athlete.ID == 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row.Athlete.Name), "НЕИЗВЕСТНЫЙ") {
			continue
		}
		sec, ok := ParseTimeString(row.TotalTime)
		if !ok {
			continue
		}
		out = append(out, ProtocolResult{
			AthleteID:    *row.Athlete.ID,
			Position:     row.Position,
			TotalTimeSec: sec,
		})
	}
	return out, nil
}

type activityProtocolJSON struct {
	Results []struct {
		TotalTime string `json:"total_time"`
		Position  int    `json:"position"`
		Athlete   struct {
			ID   *int64 `json:"id"`
			Name string `json:"name"`
		} `json:"athlete"`
	} `json:"results"`
}

type eventActivitiesJSON struct {
	Activities []struct {
		Date      string `json:"date"`
		UpdatedAt string `json:"updated_at"`
		URL       string `json:"url"`
	} `json:"activities"`
}

func parseActivityIDFromURL(raw string) (int64, error) {
	const marker = "/activities/"
	idx := strings.Index(raw, marker)
	if idx < 0 {
		return 0, fmt.Errorf("activity url: missing id")
	}
	rest := raw[idx+len(marker):]
	rest = strings.TrimSuffix(rest, ".json")
	rest = strings.Split(rest, "?")[0]
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("activity url: invalid id")
	}
	return id, nil
}