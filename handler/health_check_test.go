package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckHandler(t *testing.T) {
	expected := HealthCheck{Status: idleStatus}

	req, err := http.NewRequest("GET", "/health-check", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckHandler)

	handler.ServeHTTP(recorder, req)

	t.Run("it should respond with OK status code", func(t *testing.T) {
		if got, want := recorder.Code, http.StatusOK; got != want {
			t.Errorf("handler returned wrong status code: got %v; expected %v", got, want)
		}
	})

	t.Run("it should return a healthy status", func(t *testing.T) {
		var actual HealthCheck
		decodeErr := json.NewDecoder(recorder.Body).Decode(&actual)
		if decodeErr != nil {
			t.Errorf("could not parse JSON: %v", decodeErr)
		}
		if actual != expected {
			t.Errorf("handler returned unexpected body: got %v expected %v", recorder.Body.String(), expected)
		}
	})
}