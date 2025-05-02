// Just to test out JWT generation

package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	claims := jwt.MapClaims{
		"sub": "test",
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		fmt.Println("Error generating token:", err)
		return
	}
	fmt.Println("Generated JWT Token:", tokenString)
}