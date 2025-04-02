package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

func logRequestDetails(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request method and URL
		log.Printf("Received %s request for %s", r.Method, r.URL.Path)

		// Log all request headers
		for name, values := range r.Header {
			for _, value := range values {
				log.Printf("Header: %s = %s", name, value)
			}
		}

		// If the request method is POST, log the request body
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				log.Println("Error reading request body:", err)
				return
			}

			// Be cautious when logging large bodies; log the first 1000 characters
			log.Printf("Request body: %s", string(body[:min(len(body), 1000)])) // Limit body logged

			// Rewind the body so it can be processed by the next handler
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// Utility function to limit the size of logged body
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
