package utils

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// Claims represents the JWT claims our services expect.
type Claims struct {
	UserID uint `json:"sub"` // Assuming 'sub' holds the user ID as uint
	jwt.RegisteredClaims
}

// ValidateTokenAndGetUserID parses a JWT string, validates it, and returns the UserID.
// secretKey should be the []byte representation of your JWT secret.
func ValidateTokenAndGetUserID(tokenString string, secretKey []byte) (uint, error) {
	if tokenString == "" {
		return 0, errors.New("token string is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logrus.Errorf("Unexpected signing method in token: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		logrus.Warnf("Token parsing/validation error: %v", err)
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, errors.New("token has expired")
		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return 0, errors.New("token not valid yet")
		} else if errors.Is(err, jwt.ErrSignatureInvalid) {
            return 0, errors.New("token signature is invalid")
        }
		return 0, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.UserID == 0 {
			logrus.Warn("Token valid but UserID (sub claim) is zero")
			return 0, errors.New("invalid user identifier in token (zero)")
		}
		logrus.Infof("Token validated successfully for UserID: %d", claims.UserID)
		return claims.UserID, nil
	}

	logrus.Warn("Token deemed invalid or claims could not be parsed properly")
	return 0, errors.New("invalid token or claims")
}