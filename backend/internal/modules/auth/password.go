package auth

import "golang.org/x/crypto/bcrypt"

type PasswordHasher struct {
	cost int
}

func NewPasswordHasher(cost int) PasswordHasher {
	return PasswordHasher{cost: cost}
}

func (h PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(hash), err
}

func (h PasswordHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
