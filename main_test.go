package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var exampleTransactions = []Transaction{
	{Payer: "DANNON", Points: 1000, Timestamp: "2020-11-02T14:00:00Z"},
	{Payer: "UNILEVER", Points: 200, Timestamp: "2020-10-31T11:00:00Z"},
	{Payer: "DANNON", Points: -200, Timestamp: "2020-10-31T15:00:00Z"},
	{Payer: "MILLER COORS", Points: 10000, Timestamp: "2020-11-01T14:00:00Z"},
	{Payer: "DANNON", Points: 300, Timestamp: "2020-10-31T10:00:00Z"},
}

func TestHealthCheckHandler(t *testing.T) {
	expected := HealthCheck{Status: idleStatus}

	req, err := http.NewRequest("GET", "/health-check", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckHandler)

	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v expected %v", status, http.StatusOK)
	}

	var actual HealthCheck
	decodeErr := json.NewDecoder(recorder.Body).Decode(&actual)
	if decodeErr != nil {
		t.Errorf("could not parse JSON: %v", decodeErr)
	}
	if actual != expected {
		t.Errorf("handler returned unexpected body: got %v expected %v", recorder.Body.String(), expected)
	}
}

func TestSaveTransaction(t *testing.T) {

	for i, tr := range exampleTransactions {
		SaveTransaction(tr)
		if len(allTransactions) != i+1 {
			t.Errorf("save function didn't increase length of global transaction slice: got %v expected %v", len(allTransactions), i+1)
		}
		found := false
		for _, realTr := range allTransactions {
			if realTr == tr {
				found = true
			}
		}
		if !found {
			t.Errorf("save function didn't add transaction to global transaction slice: got %v expected %v", allTransactions, append(allTransactions, tr))
		}
	}
}
