package points

import (
	"fmt"
	"sync"
	"time"
)

// Transaction is a struct for storing how many points to associate with a payer at a given timestamp.
type Transaction struct {
	Payer         string    `json:"payer"`
	Points        int32     `json:"points"`
	Timestamp     time.Time `json:"timestamp"`
	userInitiated bool
	id            int32
}

// allTransactions is a top-scope variable acting as an in-memory database of transactions.
var allTransactions = []Transaction{}

// payerTotals is a top-scope variable acting as an in-memory summary of total points per payer.
var payerTotals = PayerTotals{}

// spentTransactions is a top-scope variable acting as an in-memory, time efficient reference map storing which Transactions have been spent.
var spentTransactions = SpendLog{}

// ByTimestamp is an alias type for a slice of Transaction objects that can be used with the sort package to improve readability.
type ByTimestamp []Transaction

// Len provides the length of a ByTimeStamp object as required by sort.Sort().
func (t ByTimestamp) Len() int { return len(t) }

// Len provides the length of a ByTimeStamp object as required by sort.Sort().
func (t ByTimestamp) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// Less returns a boolean representing wether the Transaction element at index i is "less than" the element at index j as required by sort.Sort().
func (t ByTimestamp) Less(i, j int) bool { return (t[i].Timestamp).Before(t[j].Timestamp) }

// randomUniqueIDGenerator is a type dedicated to creating random unique IDs.
type randomUniqueIDGenerator struct {
	sync.Mutex
	id int32
}

// transactionUIDs is a global instance of the randomUniqueIDGenerator type for generating Transaction struct IDs.
var transactionUIDs randomUniqueIDGenerator

// ID is a method for the randomUniqueIDGenerator struct that creates incrementing ID values.
func (rui *randomUniqueIDGenerator) ID() int32 {
	rui.Lock()
	defer rui.Unlock()

	id := rui.id
	rui.id++
	return id
}

// ResetTransactions will wipe all global variables to emulate a fresh "database".
func ResetTransactions() {
	allTransactions = []Transaction{}
	payerTotals = PayerTotals{}
	spentTransactions = SpendLog{}
}

// GetTransactions returns a slice of all the currently available Transaction objects (global allTransactions variable).
// Consider this a placeholder for a database query.
func GetTransactions() ([]Transaction, error) {
	return allTransactions, nil
}

// NewTransaction is a constructor for the Transaction struct (not user initiated) that will attempt to convert a string timestamp to a time.Time object.
// Invalid formats fo the timestamp string will result in error. RFC3339 format is expected.
//
// NOTE: This constructor will always set userInitiated to false.
func NewTransaction(payer string, points int32, timestamp string) (*Transaction, error) {
	userInit := false
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: got %v; expected time.RFC3339 format", timestamp)
	}
	result := Transaction{
		Payer:         payer,
		Points:        points,
		Timestamp:     t,
		userInitiated: userInit,
	}
	if invalidErr := result.Validate(); invalidErr != nil {
		return nil, invalidErr
	}
	return &result, nil
}

func (t *Transaction) Validate() error {
	missingAttributes := []string{}
	if t.Payer == "" {
		missingAttributes = append(missingAttributes, "payer")
	}
	if t.Points == 0 {
		missingAttributes = append(missingAttributes, "points")
	}
	if t.Timestamp.IsZero() {
		missingAttributes = append(missingAttributes, "timestamp")
	}
	if len(missingAttributes) > 0 {
		return fmt.Errorf("Validate() Invalid input - missing attributes: %v", missingAttributes)
	}
	return nil
}

// Save operates on a Transaction object, adding it to the end of the global allTransactions slice.
// Consider this a placeholder for a database query.
func (t *Transaction) Save() error {
	if err := t.Validate(); err != nil {
		return err
	}
	// Give this transaction a unique ID
	t.id = transactionUIDs.ID()
	allTransactions = append(allTransactions, *t)
	payerTotals[t.Payer] += t.Points
	return nil
}
