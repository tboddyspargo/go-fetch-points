package points

import (
	"fmt"
	"testing"
	"time"
)

func ExampleNewTransaction() {
	tr, _ := NewTransaction("PEPSI", 500, "2020-10-31T15:00:00Z")
	fmt.Println(tr)
	// Output:
	// &{PEPSI 500 2020-10-31 15:00:00 +0000 UTC false 0}
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

func testFuncForSaveTransaction(tr Transaction, lengthIncrease int, expectErr bool) func(t *testing.T) {
	return func(t *testing.T) {
		beforeTransactions, _ := GetTransactions()
		if err := tr.Save(); (expectErr && err == nil) || (!expectErr && err != nil) {
			t.Fatalf("unexpected error behavior: got %v", err)
		}

		afterTransactions, _ := GetTransactions()
		if got, want := len(afterTransactions), len(beforeTransactions)+lengthIncrease; got != want {
			t.Errorf("unexpected length of global transaction slice: got %v expected %v", got, want)
		}

		found := false
		for _, realTr := range afterTransactions {
			if realTr == tr {
				found = true
			}
		}
		if !found {
			t.Errorf("unable to find saved transaction in the database: expected %v", tr)
		}
	}
}

func TestSaveTransaction(t *testing.T) {

	tr1, _ := NewTransaction("DANNON", 1000, "2020-10-31T15:00:00Z")
	t.Run("valid transaction #1", testFuncForSaveTransaction(*tr1, 1, false))

	tr2, _ := NewTransaction("UNILEVER", 600, "2020-10-31T13:00:00Z")
	t.Run("valid transaction #1", testFuncForSaveTransaction(*tr2, 1, false))
}

func TestSpendPointsNegative(t *testing.T) {
	ResetTransactions()
	t1, _ := NewTransaction("DANNON", 100, "2020-10-31T15:00:00Z")
	t1.Save()

	if _, err := t1.SpendPoints(200); err != nil {
		t.Errorf("method should not allow a payer's balance to go below zero: got nil expected %v", err)
	}
}

func testFuncForSpendPoints(tr Transaction, points int32, expectedRemaining int32, expectedLengthIncrease int, expectErr bool) func(t *testing.T) {
	return func(t *testing.T) {
		at, _ := GetTransactions()
		beforeLength := len(at)

		pt, _ := GetPayerTotals()
		beforeTotal := pt[tr.Payer]

		st, _ := GetSpentTransactions()
		beforeSpent := st[tr.id]
		expectedActualSpend := tr.Points - beforeSpent - expectedRemaining

		spent, err := tr.SpendPoints(points)

		at, _ = GetTransactions()
		pt, _ = GetPayerTotals()
		st, _ = GetSpentTransactions()
		if got, want := pt[tr.Payer], beforeTotal-(expectedActualSpend); got != want {
			t.Errorf("method should reduce the balance for a payer: got %v expected %v", got, want)
		}
		if got, want := spent, expectedActualSpend; got != want {
			t.Errorf("method should return the number of points used from a transaction: got %v expected %v", got, want)
		}
		if got, want := st[tr.id], beforeSpent+spent; got != want {
			t.Errorf("method should update spentTransactions with amount spent: got %v expected %v", got, want)
		}
		if got, want := len(at), beforeLength+expectedLengthIncrease; got != want {
			t.Errorf("method should increase length of allTransactions when points are spent: got %v expected %v", got, want)
		}
		if (expectErr && err == nil) || (!expectErr && err != nil) {
			t.Errorf("unexpected error behavior: got %v", err)
		}
	}
}
func TestSpendPoints(t *testing.T) {
	ResetTransactions()

	tr1, _ := NewTransaction("UNILEVER", 600, "2020-10-31T13:00:00Z")
	tr1.Save()

	t.Run("spend 350 0f 600", testFuncForSpendPoints(*tr1, 350, 250, 1, false))
	t.Run("spend 200 of 250", testFuncForSpendPoints(*tr1, 200, 50, 1, false))
	t.Run("spend 500 of 50", testFuncForSpendPoints(*tr1, 500, 0, 1, false))
}
