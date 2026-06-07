package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type closeFunc func() error

func initializeLogger() (*slog.Logger, closeFunc, error) {
	logfile := os.Getenv("LINKO_LOG_FILE")

	if logfile != "" {
		file, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("couldnt open the logfile: %v", err)
		}
		buffered := bufio.NewWriterSize(file, 8192)
		multiWriter := io.MultiWriter(os.Stderr, buffered)
		logger := slog.New(slog.NewTextHandler(multiWriter, nil))

		closing := func() error {
			err := buffered.Flush()
			if err != nil {
				return err
			}
			err = file.Close()
			return err
		}
		return logger, closing, nil
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return logger, func() error { return nil }, nil

}
