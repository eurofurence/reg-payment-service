package logging

import (
	"context"
	"os"

	"github.com/eurofurence/reg-payment-service/internal/common"
	"github.com/rs/zerolog"
)

type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})

	// expected to terminate the process
	Fatal(format string, v ...interface{})
}

type loggingWrapper struct {
	logger *zerolog.Logger
}

func (l *loggingWrapper) Debug(format string, v ...interface{}) {
	l.logger.Debug().Msgf(format, v...)
}

func (l *loggingWrapper) Info(format string, v ...interface{}) {
	l.logger.Info().Msgf(format, v...)
}

func (l *loggingWrapper) Warn(format string, v ...interface{}) {
	l.logger.Warn().Msgf(format, v)
}

func (l *loggingWrapper) Error(format string, v ...interface{}) {
	l.logger.Error().Msgf(format, v...)
}

// expected to terminate the process
func (l *loggingWrapper) Fatal(format string, v ...interface{}) {
	l.logger.Fatal().Msgf(format, v...)
}

// context key with a separate type, so no other package has a chance of accessing it
type key int

// the value actually doesn't matter, the type alone will guarantee no package gets at this context value
const LoggerKey key = 0

// var defaultLogger = createLogger("00000000")

// func createLogger(requestId string) Logger {
// 	return &consolelogging.ConsoleLoggingImpl{RequestId: requestId}
// }

// func CreateContextWithLoggerForRequestId(ctx context.Context, requestId string) context.Context {
// 	return context.WithValue(ctx, loggerKey, createLogger(requestId))
// }

// you should only use this when your code really does not belong to request processing.
// otherwise be a good citizen and do pass down the context, so log output can be associated with
// the request being processed!
// func NoCtx() Logger {
// 	return defaultLogger
// }

// whenever processing a specific request, use this and give it the context.
// func Ctx(ctx context.Context) Logger {
// 	logger, ok := ctx.Value(loggerKey).(Logger)
// 	if !ok {
// 		// better than no logger at all
// 		return defaultLogger
// 	}
// 	return logger
// }

func LoggerFromContext(ctx context.Context) Logger {
	logger, ok := ctx.Value(LoggerKey).(Logger)
	if !ok {
		return NewLogger()
	}

	return logger
}

func NewLogger() Logger {
	logger := zerolog.New(os.Stdout).
		With().
		Str("App", common.ApplicationName).
		Timestamp().
		Logger()

	return &loggingWrapper{
		logger: &logger,
	}
}

func NewNoopLogger() Logger {
	return &noopLogger{}
}

type noopLogger struct {
}

func (l *noopLogger) Debug(format string, v ...interface{}) {
}

func (l *noopLogger) Info(format string, v ...interface{}) {
}

func (l *noopLogger) Warn(format string, v ...interface{}) {
}

func (l *noopLogger) Error(format string, v ...interface{}) {
}

// expected to terminate the process
func (l *noopLogger) Fatal(format string, v ...interface{}) {
}
