package airbyte

import (
	"bufio"
	"encoding/json"
	"io"
)

// LogScanner
type LogScanner struct {
	scanner *bufio.Scanner
	Line    logLine
}

// NewLogScanner returns an initialized Airbyte log scanner.
func NewLogScanner(r io.Reader) *LogScanner {
	return &LogScanner{
		scanner: bufio.NewScanner(r),
	}
}

func (j *LogScanner) Scan() bool {
	for {
		if ok := j.scanner.Scan(); !ok {
			return false
		}

		var data logLine
		err := json.Unmarshal(j.scanner.Bytes(), &data)
		// not all lines are JSON. don't propogate errors, just include the full line.
		if err != nil {
			j.Line = logLine{Message: j.scanner.Text()}
		} else {
			j.Line = data
		}

		return true
	}
}

func (j *LogScanner) Err() error {
	return j.scanner.Err()
}

/*
	{
	  "timestamp": 1734712334950,
	  "message": "Unable to bootstrap Airbyte environment.",
	  "level": "ERROR",
	  "logSource": "platform",
	  "caller": {
	    "className": "io.airbyte.bootloader.Application",
	    "methodName": "main",
	    "lineNumber": 28,
	    "threadName": "main"
	  },
	  "throwable": {
	    "cause": {
	      "cause": null,
	      "stackTrace": [
	        {
	          "cn": "io.airbyte.bootloader.Application",
	          "ln": 25,
	          "mn": "main"
	        }
	      ],
	      "message": "Unable to connect to the database.",
	      "suppressed": [],
	      "localizedMessage": "Unable to connect to the database."
	    },
	    "stackTrace": [
	      {
	        "cn": "io.airbyte.bootloader.Application",
	        "ln": 25,
	        "mn": "main"
	      }
	    ],
	    "message": "Database availability check failed.",
	    "suppressed": [],
	    "localizedMessage": "Database availability check failed."
	  }
	}
*/
type logLine struct {
	Timestamp int64         `json:"timestamp"`
	Message   string        `json:"message"`
	Level     string        `json:"level"`
	LogSource string        `json:"logSource"`
	Caller    *logCaller    `json:"caller"`
	Throwable *logThrowable `json:throwable`
}

type logCaller struct {
	ClassName  string `json:"className"`
	MethodName string `json:"methodName"`
	LineNumber int    `json:"lineNumber"`
	ThreadName string `json:"threadName"`
}

type logStackElement struct {
	ClassName  string `json:"cn"`
	LineNumber int    `json:"ln"`
	MethodName string `json:"mn"`
}

type logThrowable struct {
	Cause      *logThrowable     `json:"cause"`
	Stacktrace []logStackElement `json:"stackTrace"`
	Message    string            `json:"message"`
}
