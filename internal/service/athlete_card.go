package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/s95run/cup/internal/s95client"
)

const athleteCardVersion = "cup-athlete-v1"

var (
	ErrAthleteCardInvalid = errors.New("invalid athlete card")
	ErrAthleteCardExpired = errors.New("athlete card expired")
)

// AthleteCardTTL is how long a signed search hit remains valid for roster submission.
const AthleteCardTTL = 24 * time.Hour

type SignedAthlete struct {
	ID                int64  `json:"id"`
	Code              int64  `json:"code"`
	Name              string `json:"name"`
	Gender            string `json:"gender"`
	HomeEventCode     string `json:"home_event_code"`
	HomeEventName     string `json:"home_event_name"`
	ClubName          string `json:"club_name"`
	RunsCount         int    `json:"runs_count"`
	VolunteeringCount int    `json:"volunteering_count"`
	Exp               int64  `json:"exp"`
	Sig               string `json:"sig"`
}

type athleteCardPayload struct {
	V             string `json:"v"`
	ID            int64  `json:"id"`
	Code          int64  `json:"code"`
	Name          string `json:"name"`
	Gender        string `json:"gender"`
	HomeEventCode string `json:"home_event_code"`
	HomeEventName string `json:"home_event_name"`
	ClubName      string `json:"club_name"`
	RunsCount     int    `json:"runs_count"`
	VolCount      int    `json:"volunteering_count"`
	Exp           int64  `json:"exp"`
}

func SignAthleteCard(secret string, hit s95client.AthleteSearchHit, now time.Time) SignedAthlete {
	exp := now.Add(AthleteCardTTL).Unix()
	payload := athleteCardPayload{
		V: athleteCardVersion, ID: hit.ID, Code: hit.Code, Name: hit.Name, Gender: hit.Gender,
		HomeEventCode: hit.HomeEventCode, HomeEventName: hit.HomeEventName, ClubName: hit.ClubName,
		RunsCount: hit.RunsCount, VolCount: hit.VolunteeringCount, Exp: exp,
	}
	sig := signAthletePayload(secret, payload)
	return SignedAthlete{
		ID: hit.ID, Code: hit.Code, Name: hit.Name, Gender: hit.Gender,
		HomeEventCode: hit.HomeEventCode, HomeEventName: hit.HomeEventName, ClubName: hit.ClubName,
		RunsCount: hit.RunsCount, VolunteeringCount: hit.VolunteeringCount, Exp: exp, Sig: sig,
	}
}

func VerifyAthleteCard(secret string, card SignedAthlete, now time.Time) (s95client.AthleteSearchHit, error) {
	if card.ID == 0 || card.Sig == "" {
		return s95client.AthleteSearchHit{}, ErrAthleteCardInvalid
	}
	if card.Exp > 0 && now.Unix() > card.Exp {
		return s95client.AthleteSearchHit{}, ErrAthleteCardExpired
	}
	payload := athleteCardPayload{
		V: athleteCardVersion, ID: card.ID, Code: card.Code, Name: card.Name, Gender: card.Gender,
		HomeEventCode: card.HomeEventCode, HomeEventName: card.HomeEventName, ClubName: card.ClubName,
		RunsCount: card.RunsCount, VolCount: card.VolunteeringCount, Exp: card.Exp,
	}
	if !hmac.Equal([]byte(card.Sig), []byte(signAthletePayload(secret, payload))) {
		return s95client.AthleteSearchHit{}, ErrAthleteCardInvalid
	}
	return s95client.AthleteSearchHit{
		ID: card.ID, Code: card.Code, Name: card.Name, Gender: card.Gender,
		HomeEventCode: card.HomeEventCode, HomeEventName: card.HomeEventName, ClubName: card.ClubName,
		RunsCount: card.RunsCount, VolunteeringCount: card.VolunteeringCount,
	}, nil
}

func signAthletePayload(secret string, payload athleteCardPayload) string {
	raw, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func SignStoredMember(secret string, id int64, name, gender, homeEventName, club string, runs, vols int, now time.Time) SignedAthlete {
	code := int64(770_000_000 + id)
	return SignAthleteCard(secret, s95client.AthleteSearchHit{
		ID: id, Code: code, Name: name, Gender: gender,
		HomeEventName: homeEventName, ClubName: club,
		RunsCount: runs, VolunteeringCount: vols,
	}, now)
}