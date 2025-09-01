package log

// Log defines a minimal logging interface. This is a stub.
type Log interface{}

// logger is a no-op implementation of Log.
type logger struct{}

// Config holds options for the logger.
type Config struct{}

// New creates a new logger instance.
func New(config Config) (Log, error) {
    return &logger{}, nil
}

// Info logs an informational message.
func (l *logger) Info(format string, args ...any)  {}
// Error logs an error message.
func (l *logger) Error(format string, args ...any) {}
// Debug logs a debug message.
func (l *logger) Debug(format string, args ...any) {}
// Trace logs a trace-level message.
func (l *logger) Trace(format string, args ...any) {}
