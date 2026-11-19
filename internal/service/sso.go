package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrSSOInvalid = errors.New("invalid sso token")
	ErrSSOExpired = errors.New("sso token expired")
	ErrSSOReplay  = errors.New("sso token already used")
)

type SSOPayload struct {
	UserID        int64  `json:"user_id"`
	AthleteID     int64  `json:"athlete_id"`
	Name          string `json:"name"`
	Gender        string `json:"gender"`
	HomeEventCode string `json:"home_event_code"`
	JTI           string `json:"jti"`
	State         string `json:"state"`
	Exp           int64  `json:"exp"`
}

func VerifySSOToken(secret, tokenB64 string, now time.Time) (*SSOPayload, error) {
	raw, err := base64.RawURLEncoding.DecodeString(tokenB64)
	if err != nil {
		return nil, ErrSSOInvalid
	}
	var outer struct {
		Payload string `json:"p"`
		Sig     string `json:"s"`
	}
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, ErrSSOInvalid
	}
	if !hmac.Equal([]byte(outer.Sig), []byte(signHMAC(secret, outer.Payload))) {
		return nil, ErrSSOInvalid
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(outer.Payload)
	if err != nil {
		return nil, ErrSSOInvalid
	}
	var p SSOPayload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return nil, ErrSSOInvalid
	}
	if p.Exp > 0 && now.Unix() > p.Exp {
		return nil, ErrSSOExpired
	}
	if p.JTI == "" || p.AthleteID == 0 {
		return nil, ErrSSOInvalid
	}
	return &p, nil
}

func signHMAC(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func ConstantTimeAPIKey(got, want string) bool {
	if got == "" || want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func FormatRegistrationClose(stageDate time.Time, deadline time.Duration) time.Time {
	return stageDate.Add(-deadline)
}

func Slugify(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+'a'-'A')
		case r == ' ' || r == '-' || r == '_':
			out = append(out, '-')
		}
	}
	if len(out) == 0 {
		return fmt.Sprintf("team-%d", time.Now().Unix())
	}
	return string(out)
}
