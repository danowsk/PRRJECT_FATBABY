package feedserver

type Logger interface {
	Printf(format string, args ...any)
}
