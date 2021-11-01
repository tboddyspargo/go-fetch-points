package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var exampleTransactions = []Transaction{
	{Payer: "DANNON", Points: 1000, Timestamp: "2020-11-02T14:00:00Z", awarded: true},
	{Payer: "UNILEVER", Points: 200, Timestamp: "2020-10-31T11:00:00Z", awarded: true},
	{Payer: "DANNON", Points: -200, Timestamp: "2020-10-31T15:00:00Z", awarded: true},
	{Payer: "MILLER COORS", Points: 10000, Timestamp: "2020-11-01T14:00:00Z", awarded: true},
	{Payer: "DANNON", Points: 300, Timestamp: "2020-10-31T10:00:00Z", awarded: true},
}

// resetTransactions is used at the beginning of each test to ensure that it starts with an empty "database"
func resetTransactions() {
	allTransactions = []Transaction{}
	payerTotals = PayerTotals{}
	spentTransactions = SpendLog{}
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

func TestSave(t *testing.T) {
	resetTransactions()
	startTransactions := allTransactions
	tr1 := Transaction{Payer: "DANNON", Points: 1000, Timestamp: "2020-10-31T15:00:00Z", awarded: true}
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
		t.Errorf("save function didn't add transaction to global transaction slice: got %v expected %v", allTransactions, append(allTransactions, tr1))
	}

	midTransactions := allTransactions

	tr2 := Transaction{Payer: "UNILEVER", Points: 600, Timestamp: "2020-10-31T13:00:00Z", awarded: true}
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
		t.Errorf("save function didn't add transaction to global transaction slice: got %v expected %v", allTransactions, append(allTransactions, tr2))
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

	for _, tr := range exampleTransactions {
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
	t1 := Transaction{Payer: "DANNON", Points: 100, Timestamp: "2020-10-31T15:00:00Z", awarded: true}
	t1.Save()
	_, spendErr := t1.SpendPoints(200)
	if spendErr != nil {
		t.Errorf("method should not allow a payer's balance to go below zero: got nil expected %v", spendErr)
	}
}

func TestSpendPoints(t *testing.T) {
	resetTransactions()

	tr1 := Transaction{Payer: "DANNON", Points: 1000, Timestamp: "2020-10-31T15:00:00Z", awarded: true}
	tr1.Save()

	tr2 := Transaction{Payer: "UNILEVER", Points: 600, Timestamp: "2020-10-31T13:00:00Z", awarded: true}
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
