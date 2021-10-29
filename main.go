package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
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
// NOTE: this slice only represents AVAILABLE points. i.e. points that have not yet been spent by the user.
// When points are used, they are removed from this slice to help with performance and to prevent points from being spent more than once.
var allTransactions = []Transaction{}

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

// HealthCheck is a struct for representing the health status of the web service.
type HealthCheck struct {
	Status int `json:"status"`
}

// Transaction is a struct for storing how many points to associate with a payer at a given timestamp.
type Transaction struct {
	Payer     string `json:"payer"`
	Timestamp string `json:"timestamp"`
	Points    int32  `json:"points"`
}

// Spend is a struct that stores the number of points that a user requests to spend.
type Spend struct {
	Points int32 `json:"points"`
}

// Storing payer totals as a map allows O(1) read and update times.
type PayerTotals map[string]int32

// PayerBalance is a struct for storing the number of points associated with a payer.
type PayerBalance struct {
	Payer  string `json:"payer"`
	Points int32  `json:"points"`
}

// GetPayerTotalsMap converts individual transactions into point totals grouped by payer.
func GetPayerTotalsMap(transactions []Transaction) (PayerTotals, error) {
	var result = PayerTotals{}
	if transactions == nil {
		var err error
		transactions, err = GetTransations()
		if err != nil {
			return result, err
		}
	}
	for _, t := range transactions {
		result[t.Payer] += t.Points
	}
	return result, nil
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
func GetTotalPoints(pt PayerTotals) (int32, error) {
	var total int32
	if pt == nil {
		var err error
		pt, err = GetPayerTotalsMap(nil)
		if err != nil {
			return total, err
		}
	}
	for _, v := range pt {
		total += v
	}
	return total, nil
}

// GetTransactions returns a slice of all the currently available Transaction objects (global allTransactions variable).
func GetTransations() ([]Transaction, error) {
	return allTransactions, nil
}

// SaveTransaction appends a new Transaction object to the end of the global allTransactions slice.
func SaveTransaction(t Transaction) error {
	allTransactions = append(allTransactions, t)
	InfoLogger.Println("Added a new transaction: ", t)
	InfoLogger.Println("Total Transactions: ", len(allTransactions))
	return nil
}

// DeleteTransaction removes a Transaction object from the global allTransactions slice.
// NOTE: If Transactions are sensitive enough to require retention even after they have been used,
// then we could add new transactions indicating how many points to remove from a payer at the time of spending.
// OR
// we could keep a record of all "used" transactions so that those points could be ignored at the next spending request.
//
// We need to be careful of performance, however. In the current implementation, only unused points are retained in the allTransactions variable.
// That keeps performance reasonably fast, whereas if we retained all information, we'd want to insert keep track of "markers" that helped us identify
// used and unused points.
// WARNING: This deletion logic uses the == operator. This may not be precise enough for the sensitivity of the data. We should make sure that the equality check is precise enough to identify the correct Transaction every time.
func DeleteTransaction(t Transaction) error {
	indexToRemove := 0
	for ; indexToRemove < len(allTransactions); indexToRemove++ {
		if allTransactions[indexToRemove] == t {
			InfoLogger.Println("deleting Transaction", allTransactions[indexToRemove])
			break
		}
	}
	allTransactions = append(allTransactions[:indexToRemove], allTransactions[indexToRemove+1:]...)
	return nil
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
func AddTransactionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var t Transaction

		if jsonParseErr := json.NewDecoder(r.Body).Decode(&t); jsonParseErr != nil {
			ErrorLogger.Println("unable to parse POSTed JSON as Transaction object", jsonParseErr)
			http.Error(w, jsonParseErr.Error(), http.StatusBadRequest)
			return
		}

		if saveErr := SaveTransaction(t); saveErr != nil {
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
		transactions, getErr := GetTransations()
		if getErr != nil {
			ErrorLogger.Println(fmt.Errorf("unable to retrieve transactions: %v", getErr))
			http.Error(w, getErr.Error(), http.StatusNotFound)
			return
		}

		payerPoints, _ := GetPayerTotalsMap(transactions)

		resultBytes, parseErr := json.Marshal(PayerTotalsToPayerBalances(payerPoints))
		if parseErr != nil {
			ErrorLogger.Println(fmt.Errorf("could not convert object to JSON: %v", parseErr))
			http.Error(w, parseErr.Error(), http.StatusInternalServerError)
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

// SpendPointsHandler provides an http action for spending points agnostic of which payer will be responsible for them.
// The body of the request is expected to contain a "points" attribute indicating how many points the user would like to spend.
// Points will be spent in order of oldest to most recent and points will not be spent that bring the balance associated with a particular payer below zero.
// The response will be in the form of a JSON object representing how many points were used from each payer to satisfy the request.
func SpendPointsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var desiredSpend Spend

		err := json.NewDecoder(r.Body).Decode(&desiredSpend)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		transactions, getErr := GetTransations()
		if getErr != nil {
			return
		}
		sort.Sort(byTimestamp(transactions))
		InfoLogger.Println(transactions)

		payerPoints, _ := GetPayerTotalsMap(transactions)
		available, _ := GetTotalPoints(payerPoints)

		if available < desiredSpend.Points {
			spendErr := fmt.Errorf("insufficient points; requested: %v; available: %v", desiredSpend.Points, available)
			ErrorLogger.Println(spendErr)
			http.Error(w, spendErr.Error(), http.StatusBadRequest)
			return
		}

		var remainingToSpend int32 = desiredSpend.Points
		var pointsToSpend int32
		spentPayerPoints := PayerTotals{}
		for _, t := range transactions {
			if remainingToSpend <= 0 {
				break
			}
			// spend all remaining points unless this transaction doesn't have enough to cover what remains to be spent.
			pointsToSpend = remainingToSpend
			if t.Points < pointsToSpend {
				pointsToSpend = t.Points
			}

			// if using these points won't cause the payer to go negative,
			if (payerPoints[t.Payer] - pointsToSpend) >= 0 {

				remainingToSpend -= pointsToSpend
				payerPoints[t.Payer] -= pointsToSpend
				spentPayerPoints[t.Payer] -= pointsToSpend

				if pointsToSpend < t.Points {
					// If these points aren't all used, create a new Transaction object that reflects the points remaining from the original transaction.
					// The timestamp will be retained from the original, so the logic of "use oldest points first" will continue to be respected.
					// NOTE: the original transaction will be deleted. This may not be desireable.
					SaveTransaction(Transaction{Payer: t.Payer, Points: (t.Points - pointsToSpend), Timestamp: t.Timestamp})
				}
				// to avoid removing elements from source array while looping over it, defer the deletion
				// we could also iterate backwards through the indices to handle this differently.
				defer DeleteTransaction(t)
			}
		}

		resultBytes, parseErr := json.Marshal(PayerTotalsToPayerBalances(spentPayerPoints))
		if parseErr != nil {
			ErrorLogger.Println(fmt.Errorf("could not convert object to JSON: %v", parseErr))
			http.Error(w, parseErr.Error(), http.StatusInternalServerError)
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
