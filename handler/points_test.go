package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tboddyspargo/fetch/points"
)

func testAddTransactionHandlerWithArgs(input string, expectedStatusCode int) func(t *testing.T) {
	return func(t *testing.T) {
		input := json.RawMessage(input)
		inputJSON, mErr := json.Marshal(input)
		if mErr != nil {
			t.Fatal(mErr)
		}

		req, err := http.NewRequest("POST", "/transactions", bytes.NewReader(inputJSON))
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		handler := http.HandlerFunc(AddTransactionHandler)
		handler.ServeHTTP(recorder, req)

		r := recorder.Result()
		if got, want := r.StatusCode, expectedStatusCode; got != want {
			t.Errorf("input %v should return Created status code: got %v ; expected %v", string(input), got, want)
		}

	}
}
func TestAddTransactionHandler(t *testing.T) {
	t.Run("valid", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": 500, "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusCreated))

	t.Run("invalid object attributes missing payer", testAddTransactionHandlerWithArgs(`{ "payer": "PFIZER", "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))
	t.Run("invalid object attributes missing points", testAddTransactionHandlerWithArgs(`{ "points": 500, "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))
	t.Run("invalid object attributes missing timestamp", testAddTransactionHandlerWithArgs(`{ "payer": "PFIZER", "points": 500 }`, http.StatusBadRequest))
	t.Run("invalid object attributes missing all", testAddTransactionHandlerWithArgs(`{  }`, http.StatusBadRequest))

	t.Run("invalid payer empty", testAddTransactionHandlerWithArgs(`{ "payer": "", "points": 500, "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))
	t.Run("invalid payer int", testAddTransactionHandlerWithArgs(`{ "payer": 10, "points": 500, "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))
	t.Run("invalid payer object", testAddTransactionHandlerWithArgs(`{ "payer": {}, "points": 500, "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))
	t.Run("invalid payer array", testAddTransactionHandlerWithArgs(`{ "payer": [1,2,3,4], "points": 500, "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))

	t.Run("invalid points string", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": "MANY", "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))
	t.Run("invalid points array", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": [1,2,3,4], "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))
	t.Run("invalid points null", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": null, "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))
	t.Run("invalid points object", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": {}, "timestamp": "2020-11-02T14:00:00Z" }`, http.StatusBadRequest))

	t.Run("invalid timestamp short format", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": 500, "timestamp": "Mar 14, 2019" }`, http.StatusBadRequest))
	t.Run("invalid timestamp human name", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": 500, "timestamp": "Mark" }`, http.StatusBadRequest))
	t.Run("invalid timestamp array", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": 500, "timestamp": [1,2,3,4] }`, http.StatusBadRequest))
	t.Run("invalid timestamp int", testAddTransactionHandlerWithArgs(`{ "payer": "DANNON", "points": 500, "timestamp": 42 }`, http.StatusBadRequest))
}

func TestSpendPointsHandler(t *testing.T) {
	points.ResetTransactions()

	t1, _ := points.NewTransaction("DANNON", 1000, "2020-11-02T14:00:00Z")
	t2, _ := points.NewTransaction("UNILEVER", 200, "2020-10-31T11:00:00Z")
	t3, _ := points.NewTransaction("DANNON", -200, "2020-10-31T15:00:00Z")
	t4, _ := points.NewTransaction("MILLER COORS", 10000, "2020-11-01T14:00:00Z")
	t5, _ := points.NewTransaction("DANNON", 300, "2020-10-31T10:00:00Z")
	exampleTransactions := []points.Transaction{*t1, *t2, *t3, *t4, *t5}

	var expected = []points.PayerBalance{
		{Payer: "DANNON", Points: -100},
		{Payer: "UNILEVER", Points: -200},
		{Payer: "MILLER COORS", Points: -4700},
	}
	var expectedPayerTotals = points.PayerTotals{
		"MILLER COORS": 5300,
		"DANNON":       1000,
	}

	for _, tr := range exampleTransactions {
		tr.Save()
	}

	spendBytes, _ := json.Marshal(points.SpendRequest{Points: 5000})
	req, err := http.NewRequest("POST", "/spend", bytes.NewReader(spendBytes))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(SpendPointsHandler)
	handler.ServeHTTP(recorder, req)

	if got, want := recorder.Result().StatusCode, http.StatusOK; got != want {
		t.Fatal(fmt.Errorf("handler returned unexpected status code: got %v; want %v", got, want))
	}

	var actual []points.PayerBalance
	if err := json.NewDecoder(recorder.Body).Decode(&actual); err != nil {
		t.Fatal(fmt.Errorf("handler wasn't able to parse JSON response: got %v; error: %v", "", err))
	}

	if got, want := len(actual), len(expected); got != want {
		t.Errorf("handler didn't spend points as expected: got %v expected %v", got, want)
	}
	pt, _ := points.GetPayerTotals()
	for p, expectedTotal := range expectedPayerTotals {
		got, ok := pt[p]
		if want := expectedTotal; !ok || got != want {
			t.Errorf("handler didn't update payer balances as expected for %v: got %v expected %v", p, got, want)
		}
	}
}
