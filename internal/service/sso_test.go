package service

import (
	"testing"
	"time"
)

func TestVerifySSOToken_rejectsGarbage(t *testing.T) {
	_, err := VerifySSOToken("secret", "not-a-token", time.Now())
	if err != ErrSSOInvalid {
		t.Fatalf("got %v", err)
	}
}
