package auth

import (
	"github.com/alexedwards/argon2id"
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
