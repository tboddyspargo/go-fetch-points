package points

import (
	"errors"
	"fmt"
	"time"
)

// SpendRequest is a struct that stores the number of points that a user requests to spend.
type SpendRequest struct {
	Points int32 `json:"points"`
}

// SpendLog is an alias type for a map with transaction id keys and the number of points used from each transaction.
type SpendLog map[int32]int32

// GetSpentTransactions returns a map of Transaction ids to int32 representing how much of each Transaction has been spent.
func GetSpentTransactions() (SpendLog, error) {
	return spentTransactions, nil
}

// SetSpentTransactions updates the SpendLog for a given transaction.
func SetSpentTransactions(t Transaction, newAmount int32) {
	spentTransactions[t.id] = newAmount
}

// TODO: implement a function that spends points from across all transactions

// SpendPoints operates on a transaction and removes as many available points as possible to cover the requested amount.
// Valid transactions have been awarded by payers (not initiated by users).
// As long as it will not cause a payer's balance to go below zero, a new transaction will be added to the log indicating how many points were spent.
// The number of points actually spent will be returned.
// Consider this a placeholder for a series of database queries.
func (t *Transaction) SpendPoints(points int32) (int32, error) {
	var actualSpent int32 = 0
	toSpend := points
	available := t.Points

	// If this transaction represents points spent by the user, they cannot be spent.
	if t.userInitiated {
		awdErr := errors.New("SpendPoints() this transaction refers to spent points. You cannot spend points that have already been spent")
		return actualSpent, awdErr
	}
	// Check to see how many points from this transaction have already been used. Update the amount of points available from it.
	if spent, ok := spentTransactions[t.id]; ok {
		// Don't continue if all of these points have already been spent.
		if spent == t.Points {
			spentErr := fmt.Errorf("SpendPoints() these points have already been spent. original points: %v, spent: %v", t.Points, spent)
			return actualSpent, spentErr
		}
		available -= spent
	}
	// If this transaction doesn't have sufficient points to cover the requested amount, only spend what is available.
	if available < toSpend {
		toSpend = available
	}
	// If spending these points would bring this payer's balance below zero, don't spend them and return 0 as the number of points spent.
	if payerTotals[t.Payer]-toSpend < 0 {
		negErr := fmt.Errorf("SpendPoints() cannot spend points if it would cause a payer's balance to go below zero. available: %v, requested spend: %v", payerTotals[t.Payer], toSpend)
		return actualSpent, negErr
	}
	// Create a new Transaction to register these spent points.
	// Note that these points are being spent by the user, not awarded by a payer.
	newT := Transaction{Payer: t.Payer, Points: -toSpend, Timestamp: time.Now(), userInitiated: true}
	if saveErr := newT.Save(); saveErr != nil {
		// If this new transaction is invalid, simply return 0 - the amount spent from the original transaction.
		return actualSpent, saveErr
	}
	actualSpent = toSpend
	spentTransactions[t.id] += actualSpent
	return actualSpent, nil
}
