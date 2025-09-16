package api

import (
	"encoding/json"
	"log"
	"net/http"
)

func OKResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("error encoding response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Ideally we should be checking the error status and returning more
	// generic error messages of internal server errors for security purposes.
	// We can then use logs to keep track of the error details. I'm adding a simple log
	// to ensure we are also able to trace the error in the standard output
	log.Print(message)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("error encoding error response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
