package utils

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
)

func GenerateVerificationCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("code length must be positive")
	}
	const digits = "0123456789"
	buffer := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, buffer)
	if err != nil {
		log.Printf("Error generating random bytes for code: %v", err)
		return "", fmt.Errorf("failed to generate random code: %w", err)
	}

	for i := 0; i < length; i++ {
		buffer[i] = digits[int(buffer[i])%len(digits)]
	}

	return string(buffer), nil
}