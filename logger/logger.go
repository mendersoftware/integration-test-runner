package logger

// RequestLogger is the request logger interface
type RequestLogger interface {
	Push(string)
	Get() []string
	Clear()
}

// RequestLoggerObject is an object implementing the request logger
type RequestLoggerObject struct {
	logs []string
}

// NewRequestLogger returns a new request logger
func NewRequestLogger() RequestLogger {
	return &RequestLoggerObject{
		logs: []string{},
	}
}

// Push pushes a new log to the logger
func (r *RequestLoggerObject) Push(msg string) {
	r.logs = append(r.logs, msg)
}

// Get retrieves the logs from the logger
func (r *RequestLoggerObject) Get() []string {
	return r.logs
}

// Clear removes all the logs from the logger
func (r *RequestLoggerObject) Clear() {
	r.logs = []string{}
}

var requestLogger RequestLogger

// SetRequestLogger set the global request logger
func SetRequestLogger(rl RequestLogger) {
	requestLogger = rl
}

// GetRequestLogger get the global request logger
func GetRequestLogger() RequestLogger {
	return requestLogger
}
