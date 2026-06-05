package main

import (
	"bufio"
	"io"
	"log"
	"os"
)

func initializeLogger() (*log.Logger, error) {
	logfile := os.Getenv("LINKO_LOG_FILE")

	if logfile != "" {
		file, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			logger := log.New(os.Stderr, "", log.LstdFlags)
			return logger, err
		}
		multiWriter := io.MultiWriter(os.Stderr, bufio.NewWriterSize(file, 8192))
		logger := log.New(multiWriter, "", log.LstdFlags)
		return logger, nil
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	return logger, nil

}
