package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRespondError_WritesJSONError(t *testing.T) {
	rec := httptest.NewRecorder()

	respondError(rec, http.StatusBadRequest, "INVALID_REQUEST", "something went wrong")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"code":"INVALID_REQUEST","error":"something went wrong"}`, rec.Body.String())
}

func TestRespondError_InternalServerError(t *testing.T) {
	rec := httptest.NewRecorder()

	respondError(rec, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected failure")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"code":"INTERNAL_ERROR","error":"unexpected failure"}`, rec.Body.String())
}
