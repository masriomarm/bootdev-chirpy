package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {

	userID := uuid.New()
	secretKey := "TestKey1"

	tokenString, err := MakeJWT(userID, secretKey, time.Minute)

	if err != nil {
		t.Fatalf("Failed generating token: %v", err)
	}

	genID, err := ValidateJWT(tokenString, secretKey)

	if err != nil {
		t.Fatalf("Failed parsing token: %v", err)
	}

	if genID != userID {
		t.Errorf("ValidateJWT returned %v, want %v", genID, userID)
	}
}
