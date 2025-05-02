package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const googleVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

var recaptchaSecretKey string

func init() {
	recaptchaSecretKey = os.Getenv("RECAPTCHA_SECRET_KEY")
	if recaptchaSecretKey == "" {
		log.Println("Warning: RECAPTCHA_SECRET_KEY environment variable not set. reCAPTCHA verification will fail.")
	}
}

// RecaptchaResponse defines the structure of the response from Google.
type RecaptchaResponse struct {
	Success     bool      `json:"success"`
	ChallengeTS time.Time `json:"challenge_ts"` // Timestamp of the challenge load
	Hostname    string    `json:"hostname"`     // Hostname of the site where reCAPTCHA was solved
	ErrorCodes  []string  `json:"error-codes"`  // Optional error codes
    // Score       float64   `json:"score"`      // Only for v3
    // Action      string    `json:"action"`     // Only for v3
}


// VerifyRecaptcha sends the token to Google for verification.
func VerifyRecaptcha(token, remoteIP string) (bool, error) {
	if recaptchaSecretKey == "" {
		return false, errors.New("reCAPTCHA secret key not configured on server")
	}
    if token == "" {
        return false, errors.New("reCAPTCHA token is missing")
    }

	// Prepare the request data
	formData := url.Values{}
	formData.Set("secret", recaptchaSecretKey)
	formData.Set("response", token)
	if remoteIP != "" { // Include remote IP if available
		formData.Set("remoteip", remoteIP)
	}

	// Make the POST request
    // Use a client with timeout
    client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(googleVerifyURL, formData)
	if err != nil {
		log.Printf("ERROR: Failed to send request to reCAPTCHA verify endpoint: %v", err)
		return false, fmt.Errorf("failed to contact reCAPTCHA server: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read reCAPTCHA response body: %v", err)
		return false, fmt.Errorf("failed to read reCAPTCHA response: %w", err)
	}

	// Parse the JSON response
	var recaptchaResp RecaptchaResponse
	if err := json.Unmarshal(body, &recaptchaResp); err != nil {
		log.Printf("ERROR: Failed to parse reCAPTCHA JSON response: %v. Body: %s", err, string(body))
		return false, fmt.Errorf("failed to parse reCAPTCHA response: %w", err)
	}

    // Log details for debugging if needed
    // log.Printf("reCAPTCHA Response: Success=%v, Hostname=%s, Errors=%v",
    //     recaptchaResp.Success, recaptchaResp.Hostname, recaptchaResp.ErrorCodes)

	// Check for success and potentially error codes
	if !recaptchaResp.Success {
        errMsg := "reCAPTCHA verification failed"
        if len(recaptchaResp.ErrorCodes) > 0 {
            errMsg = fmt.Sprintf("reCAPTCHA verification failed: %v", recaptchaResp.ErrorCodes)
        }
		log.Printf("WARNING: %s (Token: %s...)", errMsg, token[:min(10, len(token))])
		return false, errors.New(errMsg) // Or return a more specific error
	}

	return true, nil // Verification successful
}

// Helper for logging token prefix
func min(a, b int) int {
    if a < b { return a }
    return b
}