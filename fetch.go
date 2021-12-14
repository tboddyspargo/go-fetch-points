package main

import (
	"flag"
	"net/http"

	h "github.com/tboddyspargo/fetch/handler"
	"github.com/tboddyspargo/fetch/log"
)

// main is the primary executor for this executable package.
// It will set up the routes and associate them with their respective handler functions.
// It also starts the http listener and logs an error if it terminates.
func main() {
	logpath := flag.String("log-path", log.DefaultLogPath, "The path (directory or file name) where logs will be written. If a directory is provided, default file name with appended date will be used - one log file per day.")
	port := flag.String("port", "8080", "The port to listen on.")
	flag.Parse()
	log.SetOutputPath(*logpath)

	http.HandleFunc("/health-check", h.HealthCheckHandler)
	http.HandleFunc("/transaction", h.AddTransactionHandler)
	http.HandleFunc("/spend", h.SpendPointsHandler)
	http.HandleFunc("/payer-points", h.PayerPointsHandler)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
