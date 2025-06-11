package utils

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecretKey []byte

func init() {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		fmt.Println("Warning: JWT_SECRET_KEY environment variable not set. Using default insecure key.")
		secret = "your-secret-key"
	}
	jwtSecretKey = []byte(secret)
}

func GenerateTokens(userID uint) (accessToken string, refreshToken string, err error) {
	// Generate Access Token
	accessClaims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Minute * 15).Unix(),
		"iat": time.Now().Unix(),
		"typ": "access",
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(jwtSecretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate Refresh Token
	refreshClaims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat": time.Now().Unix(),
		"typ": "refresh",
	}
	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(jwtSecretKey)
	if err != nil {
		// fail completely if refresh generation fails.
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing error: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return token, nil
}