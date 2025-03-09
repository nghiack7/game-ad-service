package utils

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// GenerateID generates a unique ID
func GenerateID() string {
	return uuid.New().String()
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GenerateInstanceID generates an instance ID
func GenerateInstanceID(name string) string {
	return fmt.Sprintf("%s-%s", name, RandomString(8))
}
