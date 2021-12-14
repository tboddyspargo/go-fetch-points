// Package log provides custom logging logic that is used throughout the fetch module.
package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultflags       = log.Ldate | log.Ltime | log.LUTC | log.Lmicroseconds | log.Lmsgprefix | log.Lshortfile
	defaultInfoPrefix  = "INFO: "
	defaultErrorPrefix = "ERROR: "
)

// These variables provide access to global logger objects that will be initialized on startup and used throughout the code.
var (
	InfoLogger      *log.Logger
	ErrorLogger     *log.Logger
	DefaultLogPath  = ""
	defaultFileName = fmt.Sprintf("fetch-points_%v.log", time.Now().Format("2006-01-02"))
)

// init configures loggers that will be used throughout the package to monitor behaviors.
// Messages logged will either be INFO (informational) or ERROR (errors).
// These messages can be structured and additional information added so that they can be aggregated for health and performance monitoring.
func init() {
	InfoLogger = log.New(os.Stdout, defaultInfoPrefix, defaultflags)
	ErrorLogger = log.New(os.Stderr, defaultErrorPrefix, defaultflags)
}

func SetOutputPath(s string) error {
	var file *os.File
	fi, err := os.Stat(s)
	if s == "" || (err == nil && fi.IsDir()) {
		s = filepath.Join(s, defaultFileName)
	}
	file, err = os.OpenFile(s, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	InfoLogger.SetOutput(io.MultiWriter(os.Stdout, file))
	ErrorLogger.SetOutput(io.MultiWriter(os.Stderr, file))
	return nil
}

func Infof(formatString string, values ...interface{}) {
	Info(fmt.Sprintf(formatString, values...))
}

func Info(message ...interface{}) {
	InfoLogger.Println(message...)
}

func Errorf(formatString string, values ...interface{}) {
	Error(fmt.Errorf(formatString, values...))
}

func Error(message ...interface{}) {
	ErrorLogger.Println(message...)
}

func Fatal(e error) {
	ErrorLogger.Fatal(e)
}
