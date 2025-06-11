// Acts as a reverse proxy to the AI service for category suggestions
package http

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	aiServiceBaseURL *url.URL
}

func NewAIHandler() (*AIHandler, error) {
	aiURLString := os.Getenv("AI_SERVICE_URL")
	if aiURLString == "" {
		log.Println("Warning: AI_SERVICE_URL environment variable not set. Using default http://ai-service:5000")
		aiURLString = "http://ai-service:5000"
	}

	parsedURL, err := url.Parse(aiURLString)
	if err != nil {
		log.Printf("Error parsing AI_SERVICE_URL '%s': %v", aiURLString, err)
		return nil, err
	}
	return &AIHandler{aiServiceBaseURL: parsedURL}, nil
}

// SuggestCategory proxies the request to the AI service's /suggest-category endpoint.
func (h *AIHandler) SuggestCategory(c *gin.Context) {
	proxy := httputil.NewSingleHostReverseProxy(h.aiServiceBaseURL)

	// originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = h.aiServiceBaseURL.Scheme
		req.URL.Host = h.aiServiceBaseURL.Host

		// matches AI service @app.route('/suggest-category')
		req.URL.Path = "/suggest-category"

		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				log.Printf("Error reading original request body for proxy: %v", err)
				req.Body = io.NopCloser(bytes.NewBuffer([]byte{}))
				req.ContentLength = 0
			} else {
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				req.ContentLength = int64(len(bodyBytes))
				if c.GetHeader("Content-Type") != "" {
					req.Header.Set("Content-Type", c.GetHeader("Content-Type"))
				} else {
                    req.Header.Set("Content-Type", "application/json")
                }

                // Remove Authorization header if the AI service doesn't need it
                req.Header.Del("Authorization")
			}
		}

		log.Printf("Proxying to AI service: Method=%s, URL=%s, Host=%s", req.Method, req.URL.String(), req.Host)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		log.Println("Original ACAO from AI Service:", resp.Header.Get("Access-Control-Allow-Origin"))
		resp.Header.Del("Access-Control-Allow-Origin")
		log.Println("ACAO from AI Service after delete (should be empty):", resp.Header.Get("Access-Control-Allow-Origin"))
		return nil
	}

	// Handle errors from the proxy itself
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("HTTP proxy error to AI service: %v", err)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadGateway)
		_, _ = rw.Write([]byte(`{"error": "Failed to connect to AI suggestion service"}`))
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}