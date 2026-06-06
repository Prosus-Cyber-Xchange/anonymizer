package handler

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

func respondError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{Code: code, Error: message})
}

func respondJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}
