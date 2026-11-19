package s95client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	limiter    *rate.Limiter
}

func New(baseURL, apiKey string, rps float64) *Client {
	if rps <= 0 {
		rps = 5
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		limiter:    rate.NewLimiter(rate.Limit(rps), int(rps)+1),
	}
}

type AthleteSearchHit struct {
	ID                int64  `json:"id"`
	Code              int64  `json:"code"`
	Name              string `json:"name"`
	Gender            string `json:"gender"`
	HomeEventCode     string `json:"home_event_code"`
	HomeEventName     string `json:"home_event_name"`
	ClubName          string `json:"club_name"`
	RunsCount         int    `json:"runs_count"`
	VolunteeringCount int    `json:"volunteering_count"`
}

type ProtocolResult struct {
	AthleteID    int64 `json:"athlete_id"`
	Position     int   `json:"position"`
	TotalTimeSec int   `json:"total_time_sec"`
}

type ProtocolResponse struct {
	Published  bool             `json:"published"`
	ActivityID int64            `json:"activity_id"`
	UpdatedAt  time.Time        `json:"updated_at"`
	Results    []ProtocolResult `json:"results"`
}

var ErrNotModified = fmt.Errorf("not modified")

type Event struct {
	CodeName string `json:"code_name"`
	Name     string `json:"name"`
	Town     string `json:"town"`
	Active   bool   `json:"active"`
}

type API interface {
	ListEvents(ctx context.Context) ([]Event, error)
	SearchAthletes(ctx context.Context, q string) ([]AthleteSearchHit, error)
	FetchProtocol(ctx context.Context, eventCode, date string, knownUpdatedAt *time.Time) (*ProtocolResponse, error)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body io.Reader, auth bool, dest any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if auth && c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotModified {
		return ErrNotModified
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("s95 api %s %d: %s", path, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) ListEvents(ctx context.Context) ([]Event, error) {
	var out []Event
	if err := c.doJSON(ctx, http.MethodGet, "/events.json?all=true", nil, false, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SearchAthletes(ctx context.Context, q string) ([]AthleteSearchHit, error) {
	var out struct {
		Athletes []AthleteSearchHit `json:"athletes"`
	}
	path := "/api/cup/athletes/search?" + url.Values{"q": {q}}.Encode()
	if err := c.doJSON(ctx, http.MethodGet, path, nil, true, &out); err != nil {
		return nil, err
	}
	return out.Athletes, nil
}

func (c *Client) FetchProtocol(ctx context.Context, eventCode, date string, knownUpdatedAt *time.Time) (*ProtocolResponse, error) {
	eventCode = strings.ToLower(strings.TrimSpace(eventCode))
	var listing eventActivitiesJSON
	path := "/events/" + url.PathEscape(eventCode) + ".json"
	if err := c.doJSON(ctx, http.MethodGet, path, nil, false, &listing); err != nil {
		return nil, err
	}
	var match *struct {
		Date      string
		UpdatedAt time.Time
		URL       string
	}
	for _, item := range listing.Activities {
		if item.Date != date {
			continue
		}
		updated, err := time.Parse(time.RFC3339, item.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("activity updated_at: %w", err)
		}
		copy := struct {
			Date      string
			UpdatedAt time.Time
			URL       string
		}{Date: item.Date, UpdatedAt: updated, URL: item.URL}
		match = &copy
		break
	}
	if match == nil {
		return &ProtocolResponse{Published: false}, nil
	}
	if knownUpdatedAt != nil && !match.UpdatedAt.After(*knownUpdatedAt) {
		return nil, ErrNotModified
	}
	activityID, err := parseActivityIDFromURL(match.URL)
	if err != nil {
		return nil, err
	}
	var body activityProtocolJSON
	activityPath := fmt.Sprintf("/activities/%d.json", activityID)
	if err := c.doJSON(ctx, http.MethodGet, activityPath, nil, false, &body); err != nil {
		return nil, err
	}
	results, err := parseActivityResults(body)
	if err != nil {
		return nil, err
	}
	return &ProtocolResponse{
		Published:  true,
		ActivityID: activityID,
		UpdatedAt:  match.UpdatedAt,
		Results:    results,
	}, nil
}