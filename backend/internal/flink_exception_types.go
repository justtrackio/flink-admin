package internal

// FlinkExceptionEntry represents a single exception in the Flink job exception history.
// This maps to both the RootExceptionInfo and ExceptionInfo in the Flink REST API.
type FlinkExceptionEntry struct {
	ExceptionName        string                `json:"exceptionName"`
	Stacktrace           string                `json:"stacktrace"`
	Timestamp            int64                 `json:"timestamp"`
	TaskName             string                `json:"taskName"`
	Location             string                `json:"location"`
	Endpoint             string                `json:"endpoint"`
	TaskManagerId        string                `json:"taskManagerId"`
	FailureLabels        map[string]string     `json:"failureLabels,omitempty"`
	ConcurrentExceptions []FlinkExceptionEntry `json:"concurrentExceptions,omitempty"`
}

// FlinkExceptionHistory holds the exception history for a Flink job.
type FlinkExceptionHistory struct {
	Entries   []FlinkExceptionEntry `json:"entries"`
	Truncated bool                  `json:"truncated"`
}

// FlinkJobExceptions is the response from GET /jobs/:jobid/exceptions
type FlinkJobExceptions struct {
	ExceptionHistory FlinkExceptionHistory `json:"exceptionHistory"`
}
