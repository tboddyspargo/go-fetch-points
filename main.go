package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

// These constants provide a set of private enum values representing web service status
const (
	idleStatus = iota
	busyStatus
	errorStatus
	notRunningStatus
)

// These variables provide access to global logger objects that will be initialized on startup and used throughout the code.
var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

// allTransactions is a top-scope variable acting as an in-memory database of transactions.
var allTransactions = []Transaction{}

// payerTotals is a top-scope variable acting as an in-memory summary of total points per payer.
var payerTotals = PayerTotals{}

// spentTransactions is a top-scope variable acting as an in-memory, time efficient reference map storing which Transactions have been spent.
var spentTransactions = SpendLog{}

// byTimestamp is an alias type for a slice of Transaction objects that can be used with sort to improve readability.
type byTimestamp []Transaction

// Len provides the length of a byTimeStamp object as required by sort.Sort().
func (t byTimestamp) Len() int { return len(t) }

// Len provides the length of a byTimeStamp object as required by sort.Sort().
func (t byTimestamp) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

// Less returns a boolean representing wether the element at index i in the byTimeStamp slice
// is "less than" the element at index j as required by sort.Sort().
func (t byTimestamp) Less(i, j int) bool {
	var iDate, iErr = time.Parse(time.RFC3339, t[i].Timestamp)
	var jDate, jErr = time.Parse(time.RFC3339, t[j].Timestamp)
	if iErr != nil || jErr != nil {
		ErrorLogger.Println("invalid timestamp format", t[i], t[j])
	}
	return iDate.Before(jDate)
}

// randomUniqueIDGenerator is a type dedicated to creating random unique IDs.
type randomUniqueIDGenerator struct {
	sync.Mutex
	id int32
}

// transactionUIDs is a global instance of the randomUniqueIDGenerator type for generating Transaction struct IDs.
var transactionUIDs randomUniqueIDGenerator

// ID is a method for the randomUiqueIDGenerator struct that creates incrementing ID values.
func (rui *randomUniqueIDGenerator) ID() int32 {
	rui.Lock()
	defer rui.Unlock()

	id := rui.id
	rui.id++
	return id
}

// HealthCheck is a struct for representing the health status of the web service.
type HealthCheck struct {
	Status int `json:"status"`
}

// Transaction is a struct for storing how many points to associate with a payer at a given timestamp.
type Transaction struct {
	id        int32
	awarded   bool
	Payer     string `json:"payer"`
	Timestamp string `json:"timestamp"`
	Points    int32  `json:"points"`
}

// SpendRequest is a struct that stores the number of points that a user requests to spend.
type SpendRequest struct {
	Points int32 `json:"points"`
}

// PayerTotals is an alias for a map with payer name keys and their respective point totals. It provides more efficient lookup and update speeds
type PayerTotals map[string]int32

// SpendLog is an alias type for a map with transaction id keys and the number of points used from each transaction.
type SpendLog map[int32]int32

// PayerBalance is a struct for storing the number of points associated with a payer. An array of these is the expected return type of several API routes.
type PayerBalance struct {
	Payer  string `json:"payer"`
	Points int32  `json:"points"`
}

// PayerTotalsToPayerBalances converts a PayerTotals map to a slice of PayerBalance objects, which is what the web service is expected to return.
func PayerTotalsToPayerBalances(pt PayerTotals) []PayerBalance {
	var result = []PayerBalance{}
	for k, v := range pt {
		result = append(result, PayerBalance{Payer: k, Points: v})
	}
	return result
}

// GetTotalPoints returns the sum of all points for all payers.
func GetTotalPoints() int32 {
	var total int32 = 0
	for _, v := range payerTotals {
		total += v
	}
	return total
}

// GetTransactions returns a slice of all the currently available Transaction objects (global allTransactions variable).
// Consider this a placeholder for a database query.
func GetTransactions() ([]Transaction, error) {
	return allTransactions, nil
}

// Save operates on a Transaction object, adding it to the end of the global allTransactions slice.
// Consider this a placeholder for a database query.
func (t *Transaction) Save() error {
	// Give this transaction a unique ID
	t.id = transactionUIDs.ID()
	allTransactions = append(allTransactions, *t)
	payerTotals[t.Payer] += t.Points
	InfoLogger.Println("Added a new transaction: ", *t)
	return nil
}

// SpendPoints operates on a transaction and removes as many available points as possible to cover the requested amount.
// Valid transactions have been awarded to users (are not records of users spending points)
// As long as it will not cause a payer's balance to go below zero, a new transaction will be added to the log indicating how many points were spent.
// The number of points actually spent will be returned.
// Consider this a placeholder for a series of database queries.
func (t *Transaction) SpendPoints(points int32) (int32, error) {
	var actualSpent int32 = 0
	toSpend := points
	available := t.Points

	// If these points were not awarded to the user, then skip them. This means the transaction refers to the user "consuming" his/her own points.
	if !t.awarded {
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
	// Note that they were not "awarded" to the user. The user is spending them.
	newT := Transaction{Payer: t.Payer, Points: -toSpend, Timestamp: time.Now().Format(time.RFC3339), awarded: false}
	if saveErr := newT.Save(); saveErr != nil {
		// If this new transaction is invalid, simply return 0 - the amount spent from the original transaction.
		return actualSpent, saveErr
	}
	actualSpent = toSpend
	spentTransactions[t.id] += actualSpent
	return actualSpent, nil
}

// HealthCheckHandler provides an http response representing the health status of the web service.
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		resultBytes, err := json.Marshal(HealthCheck{Status: idleStatus})
		if err != nil {
			ErrorLogger.Println(fmt.Errorf("could not convert object to JSON: %v", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resultJSON := string(resultBytes)
		InfoLogger.Println(resultJSON)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, resultJSON)
	default:
		ErrorLogger.Println(fmt.Errorf("HealthCheckHandler only supports GET requests"))
	}
}

// AddTransactionHandler provides http action for creating new Transaction records.
// The body of the request is expected to contain the relevant fields for a Transaction object.
// All Transactions created by this route are expected to be points "awarded" to a user. It should not be used for "spending" points.
func AddTransactionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var t Transaction

		// Populate the transaction object (t) from the body of the request.
		if parseErr := json.NewDecoder(r.Body).Decode(&t); parseErr != nil {
			ErrorLogger.Println("unable to parse POSTed JSON as Transaction object", parseErr)
			http.Error(w, parseErr.Error(), http.StatusBadRequest)
			return
		}

		// Assume transactions created using this API route correspond to points that have been 'awarded' to the user (as opposed to spent by the user).
		t.awarded = true

		if saveErr := t.Save(); saveErr != nil {
			ErrorLogger.Println("unable to create new transaction object", saveErr)
			http.Error(w, saveErr.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	default:
		ErrorLogger.Println(fmt.Errorf("AddTransactionHandler only supports POST requests"))
	}
}

// PayerPointsHandler provides an http response in the form of a JSON object representing the total points for every payer.
func PayerPointsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		resultBytes, mErr := json.Marshal(PayerTotalsToPayerBalances(payerTotals))
		if mErr != nil {
			ErrorLogger.Println(fmt.Errorf("could not convert object to JSON: %v", mErr))
			http.Error(w, mErr.Error(), http.StatusInternalServerError)
			return
		}
		resultJSON := string(resultBytes)
		InfoLogger.Println(resultJSON)
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, resultJSON)
	default:
		ErrorLogger.Println(fmt.Errorf("PayerPointsHandler only supports GET requests"))
	}
}

// SpendPointsHandler provides an http action for spending a specified number of points.
// The body of the request is expected to contain a "points" attribute indicating how many points the user would like to spend.
// Points will be spent in order of oldest to most recent and points will not be spent if doing so would bring the balance associated with a particular payer below zero.
// The response will be in the form of a JSON array containing objects representing how many points were used from each payer to satisfy the request.
func SpendPointsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var desiredSpend SpendRequest
		err := json.NewDecoder(r.Body).Decode(&desiredSpend)
		if err != nil {
			ErrorLogger.Printf("unable to parse json: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		totalAvailable := GetTotalPoints()
		if totalAvailable < desiredSpend.Points {
			spendErr := fmt.Errorf("insufficient points. requested: %v; available: %v", desiredSpend.Points, totalAvailable)
			ErrorLogger.Println(spendErr)
			http.Error(w, spendErr.Error(), http.StatusBadRequest)
			return
		}

		// Sort the transactions in order of oldest to newest.
		transactions, _ := GetTransactions()
		sort.Sort(byTimestamp(transactions))

		// Keep track of how many points are spent from each payer to satisfy this request.
		spentPayerPoints := PayerTotals{}

		var remainingToSpend int32 = desiredSpend.Points
		for _, t := range transactions {
			// If all requested points have been spent, we're done
			if remainingToSpend <= 0 {
				break
			}

			// Attempt to spend the points from this transaction.
			currentSpent, spendErr := t.SpendPoints(remainingToSpend)
			if spendErr != nil {
				continue
			}
			spentPayerPoints[t.Payer] -= currentSpent
			remainingToSpend -= currentSpent
		}

		resultBytes, mErr := json.Marshal(PayerTotalsToPayerBalances(spentPayerPoints))
		if mErr != nil {
			ErrorLogger.Println(fmt.Errorf("could not convert object to JSON: %v", mErr))
			http.Error(w, mErr.Error(), http.StatusInternalServerError)
			return
		}

		resultJSON := string(resultBytes)
		InfoLogger.Println(resultJSON)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, resultJSON)
	default:
		ErrorLogger.Println(fmt.Errorf("AddTransactionHandler only supports POST requests"))
	}
}

// init configures loggers that will be used throughout the package to monitor behaviors.
// Messages logged will either be INFO (informational) or ERROR (errors).
// These messages can be structured and additional information added so that they can be aggregated for health and performance monitoring.
func init() {
	filename := fmt.Sprintf("fetch-points_%v.log", time.Now().Format("2006-01-02"))
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.LUTC|log.Lmicroseconds|log.Lmsgprefix|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.LUTC|log.Lmicroseconds|log.Lmsgprefix|log.Lshortfile)
}

// main is the primary executor for this executable package.
// It will set up the routes and associate them with their respective handler functions.
// It also starts the http listener and logs an error if it terminates.
func main() {
	http.HandleFunc("/health-check", HealthCheckHandler)
	http.HandleFunc("/transactions", AddTransactionHandler)
	http.HandleFunc("/spend", SpendPointsHandler)
	http.HandleFunc("/payer-points", PayerPointsHandler)
	ErrorLogger.Fatal(http.ListenAndServe(":8080", nil))
}
