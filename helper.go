package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"boot.dev/linko/internal/linkoerr"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	pkger "github.com/pkg/errors"
	"gopkg.in/natefinch/lumberjack.v2"
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

	handlers := []slog.Handler{
		tint.NewTextHandler(os.Stderr, &tint.Options{
			Level:       slog.LevelDebug,
			ReplaceAttr: replaceAttr,
			NoColor:     !(isatty.IsCygwinTerminal(os.Stderr.Fd()) || isatty.IsTerminal(os.Stderr.Fd())),
		}),
	}

	if logfile != "" {

		logger := &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    1,
			MaxAge:     28,
			MaxBackups: 10,
			LocalTime:  false,
			Compress:   true,
		}

		handlers = append(handlers, slog.NewJSONHandler(logger, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttr,
		}))

		closing := func() error {
			err := logger.Close()
			if err != nil {
				return err
			}
			return nil
		}

		return slog.New(slog.NewMultiHandler(handlers...)), closing, nil
	}

	return slog.New(slog.NewMultiHandler(handlers...)), func() error { return nil }, nil

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
	if status == 401 || status == 403 || status == 500 {
		http.Error(w, http.StatusText(status), status)
	} else {
		http.Error(w, err.Error(), status)
	}
}

func redactIP(address string) string {

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	ip := net.ParseIP(host)

	ip = ip.To4()
	if ip == nil {
		return address
	}

	ipslice := strings.Split(ip.String(), ".")
	ipslice[len(ipslice)-1] = "x"
	ipstring := strings.Join(ipslice, ".")

	return ipstring
}
