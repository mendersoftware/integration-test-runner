package logger

import "encoding/json"

// RequestLogger is the request logger interface
type RequestLogger interface {
	Push(string)
	Write(p []byte) (n int, err error)
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

// Write parses a JSON log message and add it to the logs (io.Writer interface)
func (r *RequestLoggerObject) Write(p []byte) (n int, err error) {
	log := &struct {
		Time    string `json:"time"`
		Level   string `json:"level"`
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(p, log); err == nil {
		r.Push(log.Level + ":" + log.Message)
	}
	return len(p), nil
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
