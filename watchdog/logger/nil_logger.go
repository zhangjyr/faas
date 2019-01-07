package logger

// NullLogger - An empty logger that ignores everything
type NilLogger struct{}

// Trace - no-op
func (logger *NilLogger) Trace(format string, args ...interface{}) {}

// Debug - no-op
func (logger *NilLogger) Debug(format string, args ...interface{}) {}

// Info - no-op
func (logger *NilLogger) Info(format string, args ...interface{}) {}

// Warn - no-op
func (logger *NilLogger) Warn(format string, args ...interface{}) {}

// Warn - no-op
func (logger *NilLogger) Error(format string, args ...interface{}) {}
