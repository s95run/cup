package s95client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseTimeString(t *testing.T) {
	tests := []struct {
		in  string
		sec int
		ok  bool
	}{
		{"23:20", 1400, true},
		{"1:05:30", 3930, true},
		{"xx:xx", 0, false},
		{"", 0, false},
		{"bad", 0, false},
	}
	for _, tc := range tests {
		sec, ok := ParseTimeString(tc.in)
		if ok != tc.ok || (ok && sec != tc.sec) {
			t.Fatalf("%q => (%d,%v) want (%d,%v)", tc.in, sec, ok, tc.sec, tc.ok)
		}
	}
}

func TestFetchProtocol_cascade(t *testing.T) {
	const updated = "2026-12-05T10:00:00Z"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/events/angarskieprudy.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"activities": []map[string]string{
					{"date": "2026-12-05", "updated_at": updated, "url": "https://evil.example/activities/9001.json"},
				},
			})
		case "/activities/9001.json":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{
					{"total_time": "23:20", "position": 1, "athlete": map[string]any{"id": 42, "name": "Anna"}},
					{"total_time": "xx:xx", "position": 2, "athlete": map[string]any{"id": 43, "name": "Bob"}},
					{"total_time": "24:00", "position": 3, "athlete": map[string]any{"name": "НЕИЗВЕСТНЫЙ"}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := New(srv.URL, "key", 100)
	proto, err := client.FetchProtocol(context.Background(), "angarskieprudy", "2026-12-05", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Published || proto.ActivityID != 9001 {
		t.Fatalf("proto %+v", proto)
	}
	if len(proto.Results) != 1 || proto.Results[0].AthleteID != 42 || proto.Results[0].TotalTimeSec != 1400 {
		t.Fatalf("results %+v", proto.Results)
	}
}

func TestFetchProtocol_notModified(t *testing.T) {
	known := time.Date(2026, 12, 5, 11, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/events/x.json" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"activities": []map[string]string{
					{"date": "2026-12-05", "updated_at": "2026-12-05T10:00:00Z", "url": "/activities/1.json"},
				},
			})
			return
		}
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, "", 100)
	_, err := client.FetchProtocol(context.Background(), "x", "2026-12-05", &known)
	if err != ErrNotModified {
		t.Fatalf("got %v", err)
	}
}

func TestSearchAthletes_sumsRolesExceptOther(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"athletes": []map[string]any{{
				"id": 7, "code": 770000007, "name": "Анна", "gender": "female",
				"runs_count":           0,
				"volunteering_by_role": map[string]int{"marshal": 4, "other": 2, "": 9},
			}},
		})
	}))
	defer srv.Close()

	hits, err := New(srv.URL, "key", 100).SearchAthletes(context.Background(), "Анна")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].VolunteeringCount != 4 {
		t.Fatalf("hits %+v", hits)
	}
}

func TestFetchProtocol_missingDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"activities": []any{}})
	}))
	defer srv.Close()
	client := New(srv.URL, "", 100)
	proto, err := client.FetchProtocol(context.Background(), "x", "2026-12-05", nil)
	if err != nil || proto.Published {
		t.Fatalf("proto %+v err %v", proto, err)
	}
}
