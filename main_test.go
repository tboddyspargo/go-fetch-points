package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func exampleTransactions() []Transaction {
	t1, _ := NewTransaction("DANNON", 1000, "2020-11-02T14:00:00Z", false)
	t2, _ := NewTransaction("UNILEVER", 200, "2020-10-31T11:00:00Z", false)
	t3, _ := NewTransaction("DANNON", -200, "2020-10-31T15:00:00Z", false)
	t4, _ := NewTransaction("MILLER COORS", 10000, "2020-11-01T14:00:00Z", false)
	t5, _ := NewTransaction("DANNON", 300, "2020-10-31T10:00:00Z", false)

	return []Transaction{*t1, *t2, *t3, *t4, *t5}
}

// resetTransactions is used at the beginning of each test to ensure that it starts with an empty "database"
func resetTransactions() {
	allTransactions = []Transaction{}
	payerTotals = PayerTotals{}
	spentTransactions = SpendLog{}
}

func testFuncForValidateTransaction(tr Transaction) func(t *testing.T) {
	return func(t *testing.T) {
		err := tr.Validate()
		if err == nil {
			t.Error("got nil; expected error")
		}
	}
}

func TestValidateTransaction(t *testing.T) {
	t.Run("invalid payer empty", testFuncForValidateTransaction(Transaction{Payer: "", Points: 5, Timestamp: time.Now()}))

	t.Run("invalid points zero", testFuncForValidateTransaction(Transaction{Payer: "DANNON", Points: 0, Timestamp: time.Now()}))

	t.Run("invalid timestamp none", testFuncForValidateTransaction(Transaction{Payer: "DANNON", Points: 5}))
}

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

func TestSaveTransaction(t *testing.T) {
	resetTransactions()
	startTransactions := allTransactions
	tr1, _ := NewTransaction("DANNON", 1000, "2020-10-31T15:00:00Z", true)
	saveErr := tr1.Save()

	if saveErr != nil {
		t.Fatalf("save function returned an error: got %v expected nil", saveErr)
	} else if len(allTransactions) != len(startTransactions)+1 {
		t.Errorf("save function didn't increase length of global transaction slice: got %v expected %v", len(allTransactions), len(startTransactions)+1)
	}

	found1 := false
	for _, realTr := range allTransactions {
		if realTr.Payer == tr1.Payer && realTr.Points == tr1.Points && realTr.Timestamp == tr1.Timestamp {
			found1 = true
		}
	}
	if !found1 {
		t.Errorf("save function didn't add transaction to global transaction slice: got %v expected %v", allTransactions, append(allTransactions, *tr1))
	}

	midTransactions := allTransactions

	tr2, _ := NewTransaction("UNILEVER", 600, "2020-10-31T13:00:00Z", true)
	tr2.Save()

	if saveErr != nil {
		t.Fatalf("save function returned an error: got %v expected nil", saveErr)
	} else if len(allTransactions) != len(midTransactions)+1 {
		t.Errorf("save function didn't increase length of global transaction slice: got %v expected %v", len(allTransactions), len(midTransactions)+1)
	}

	found2 := false
	for _, realTr := range allTransactions {
		if realTr.Payer == tr2.Payer && realTr.Points == tr2.Points && realTr.Timestamp == tr2.Timestamp {
			found2 = true
		}
	}
	if !found2 {
		t.Errorf("save function didn't add transaction to global transaction slice: got %v expected %v", allTransactions, append(allTransactions, *tr2))
	}
}

func TestSpendPointsHandler(t *testing.T) {
	resetTransactions()
	var expected = []PayerBalance{
		{Payer: "DANNON", Points: -100},
		{Payer: "UNILEVER", Points: -200},
		{Payer: "MILLER COORS", Points: -4700},
	}
	var expectedPayerTotals = PayerTotals{
		"MILLER COORS": 5300,
		"DANNON":       1000,
	}

	for _, tr := range exampleTransactions() {
		tr.Save()
	}
	desiredSpend := SpendRequest{Points: 5000}
	spendBytes, _ := json.Marshal(desiredSpend)
	req, err := http.NewRequest("POST", "/spend", bytes.NewReader(spendBytes))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(SpendPointsHandler)
	handler.ServeHTTP(recorder, req)

	var actual []PayerBalance
	mErr := json.NewDecoder(recorder.Body).Decode(&actual)
	if mErr != nil {
		t.Fatal(fmt.Errorf("unable to parse JSON response: got %v; error: %v", recorder.Body, mErr))
	}

	if len(actual) != len(expected) {
		t.Errorf("handler didn't spend points as expected: got %v expected %v", actual, expected)
	}
	for p, expectedTotal := range expectedPayerTotals {
		if actualTotal, ok := payerTotals[p]; !ok || actualTotal != expectedTotal {
			t.Errorf("handler didn't update payer balances as expected for %v: got %v expected %v", p, actualTotal, expectedTotal)
		}
	}
}

func TestSpendPointsNegative(t *testing.T) {
	resetTransactions()
	t1, _ := NewTransaction("DANNON", 100, "2020-10-31T15:00:00Z", true)
	t1.Save()
	_, spendErr := t1.SpendPoints(200)
	if spendErr != nil {
		t.Errorf("method should not allow a payer's balance to go below zero: got nil expected %v", spendErr)
	}
}

func TestSpendPoints(t *testing.T) {
	resetTransactions()

	tr1, _ := NewTransaction("DANNON", 1000, "2020-10-31T15:00:00Z", true)
	tr1.Save()

	tr2, _ := NewTransaction("UNILEVER", 600, "2020-10-31T13:00:00Z", true)
	tr2.Save()

	var spend2a, remain2a int32 = 350, 250
	tr2.SpendPoints(spend2a)

	if payerTotals[tr2.Payer] != remain2a {
		t.Errorf("method should reduce the balance for a payer: got %v expected %v", payerTotals[tr2.Payer], remain2a)
	}
	if spentTransactions[tr2.id] != spend2a {
		t.Errorf("method should indicate that points were used from a transaction: got %v expected %v", spentTransactions[tr2.id], spend2a)
	}
	if len(allTransactions) != 3 {
		t.Errorf("method should increase length of allTransactions when points are spent: got %v expected %v", len(allTransactions), 3)
	}

	var spend2b, remain2b int32 = 200, 50
	tr2.SpendPoints(spend2b)

	if payerTotals[tr2.Payer] != remain2b {
		t.Errorf("method should reduce the balance for a payer: got %v expected %v", payerTotals[tr2.Payer], remain2b)
	}
	if spentTransactions[tr2.id] != spend2a+spend2b {
		t.Errorf("method should indicate that points were used from a transaction: got %v expected %v", spentTransactions[tr2.id], spend2a+spend2b)
	}
	if len(allTransactions) != 4 {
		t.Errorf("method should increase length of allTransactions when points are spent: got %v expected %v", len(allTransactions), 4)
	}

}
