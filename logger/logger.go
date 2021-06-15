package logger

type RequestLogger interface {
	Push(string)
	Get() []string
	Clear()
}

type RequestLoggerObject struct {
	logs []string
}

func NewRequestLogger() RequestLogger {
	return &RequestLoggerObject{
		logs: []string{},
	}
}

func (r *RequestLoggerObject) Push(msg string) {
	r.logs = append(r.logs, msg)
}

func (r *RequestLoggerObject) Get() []string {
	return r.logs
}

func (r *RequestLoggerObject) Clear() {
	r.logs = []string{}
}

var requestLogger RequestLogger

func SetRequestLogger(rl RequestLogger) {
	requestLogger = rl
}

func GetRequestLogger() RequestLogger {
	return requestLogger
}
