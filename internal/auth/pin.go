package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
)

func NewPIN() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func VerifyPIN(supplied, actual string) bool {
	return len(supplied) == len(actual) &&
		subtle.ConstantTimeCompare([]byte(supplied), []byte(actual)) == 1
}
