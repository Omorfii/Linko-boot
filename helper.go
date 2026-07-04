package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"boot.dev/linko/internal/linkoerr"
	pkger "github.com/pkg/errors"
)

type closeFunc func() error

type stackTracer interface {
	error
	StackTrace() pkger.StackTrace
}

type multiError interface {
	error
	Unwrap() []error
}

func initializeLogger() (*slog.Logger, closeFunc, error) {
	logfile := os.Getenv("LINKO_LOG_FILE")

	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
	})

	if logfile != "" {
		file, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("couldnt open the logfile: %v", err)
		}
		buffered := bufio.NewWriterSize(file, 8192)
		infoHandler := slog.NewJSONHandler(buffered, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
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

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == "error" {
		err, ok := a.Value.Any().(error)
		if !ok {
			return a
		}

		var attrs []slog.Attr

		if me, ok := errors.AsType[multiError](err); ok {
			var errAttrs []slog.Attr
			for i, err := range me.Unwrap() {
				errAttrs = append(errAttrs, slog.GroupAttrs(fmt.Sprintf("error_%d", i+1), linkoerr.Attrs(err)...))
			}
			attrs = append(attrs, errAttrs...)
			return slog.GroupAttrs("errors", attrs...)

		} else {

			attrs = append(attrs, slog.Attr{
				Key:   "message",
				Value: slog.StringValue(err.Error()),
			})

			if stackErr, ok := errors.AsType[stackTracer](err); ok {
				attrs = append(attrs, slog.Attr{
					Key:   "stack_trace",
					Value: slog.StringValue(fmt.Sprintf("%+v", stackErr.StackTrace())),
				})
			}

			attrs = append(attrs, linkoerr.Attrs(err)...)
			return slog.GroupAttrs("error", attrs...)
		}
	}
	return a
}

func httpError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	if logCtx, ok := ctx.Value(logContextKey).(*logContext); ok {
		logCtx.Error = err
	}
	http.Error(w, err.Error(), status)
}
