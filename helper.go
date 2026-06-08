package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

type closeFunc func() error

func initializeLogger() (*slog.Logger, closeFunc, error) {
	logfile := os.Getenv("LINKO_LOG_FILE")

	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	if logfile != "" {
		file, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("couldnt open the logfile: %v", err)
		}
		buffered := bufio.NewWriterSize(file, 8192)
		infoHandler := slog.NewTextHandler(buffered, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})

		closing := func() error {
			err := buffered.Flush()
			if err != nil {
				return err
			}
			err = file.Close()
			return err
		}

		logger := slog.New(slog.NewMultiHandler(
			debugHandler,
			infoHandler,
		))

		return logger, closing, nil
	}

	logger := slog.New(debugHandler)
	return logger, func() error { return nil }, nil

}
