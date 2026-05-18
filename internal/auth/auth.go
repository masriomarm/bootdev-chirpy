package auth

import (
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	// Hash the password using the argon2id.CreateHash function.
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	return hash, err
}
func CheckPasswordHash(password, hash string) (bool, error) {
	// : Use the argon2id.ComparePasswordAndHash
	match, err := argon2id.ComparePasswordAndHash(password, hash)

	return match, err
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	issued := jwt.NewNumericDate(time.Now().In(time.UTC))
	expires := jwt.NewNumericDate(issued.Time.Add(expiresIn))
	subject := userID.String()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: "chirpy-access", IssuedAt: issued, ExpiresAt: expires, Subject: subject})

	key := []byte(tokenSecret)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) { return []byte(tokenSecret), nil })

	if err != nil {
		return uuid.Nil, nil
	}

	userString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, nil
	}

	return uuid.Parse(userString)
}
