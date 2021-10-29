package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

// SaveTransaction appends a new Transaction object to the end of the global allTransactions slice.
func SaveTransaction(t Transaction) error {
	allTransactions = append(allTransactions, t)
	InfoLogger.Println("Added a new transaction: ", t)
	InfoLogger.Println("Total Transactions: ", len(allTransactions))
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
	ErrorLogger.Fatal(http.ListenAndServe(":8080", nil))
}
