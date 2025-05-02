package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				logrus.Errorf("Unexpected signing method: %v", token.Header["alg"])
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil {
			logrus.Warnf("Token parsing error: %v", err)
			errorMsg := "Invalid or expired token"
			if errors.Is(err, jwt.ErrTokenExpired) {
				errorMsg = "Token has expired"
			} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
				errorMsg = "Token not valid yet"
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorMsg})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// --- Extract User ID from 'sub' claim ---
			if subject, ok := claims["sub"]; ok {
				// Convert subject to uint for internal use (assuming it's stored as number)
				// Handle potential type assertion errors carefully
				var userID uint
				switch v := subject.(type) {
				case float64: // Numeric claims are often parsed as float64
					userID = uint(v)
				case uint:
					userID = v
				case int:
					userID = uint(v)
				default:
					logrus.Warnf("Invalid user ID type in token claim 'sub': %T", subject)
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user identifier in token"})
					c.Abort()
					return
				}

				if userID == 0 {
					logrus.Warn("User ID claim 'sub' is zero in token")
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user identifier in token"})
					c.Abort()
					return
				}

				// --- Set UserID in Gin Context ---
				c.Set("userID", userID) // Use a consistent key like "userID"
				logrus.Infof("Authenticated User ID: %d", userID) // Log successful auth

			} else {
				logrus.Warn("Token missing 'sub' (subject) claim")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token missing user identifier"})
				c.Abort()
				return
			}
		} else {
			logrus.Warn("Token deemed invalid or claims could not be parsed")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}