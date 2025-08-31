package log

type Log interface{}

type logger struct{}

type Config struct{}

func New(config Config) (Log, error) {
	return &logger{}, nil
}

func (l *logger) Info(format string, args ...any)  {}
func (l *logger) Error(format string, args ...any) {}
func (l *logger) Debug(format string, args ...any) {}
func (l *logger) Trace(format string, args ...any) {}
