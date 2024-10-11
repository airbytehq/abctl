package local

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// 2024-09-10 20:16:24 WARN i.m.s.r.u.Loggers$Slf4JLogger(warn):299 - [273....
var logRx = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \x1b\[(?:1;)?\d+m(?P<level>[A-Z]+)\x1b\[m (?P<msg>\S+ - .*)`)

type logLine struct {
	msg   string
	level string
}

type logScanner struct {
	scanner *bufio.Scanner
	line    logLine
}

func newLogScanner(r io.Reader) *logScanner {
	return &logScanner{
		scanner: bufio.NewScanner(r),
		line: logLine{
			msg:   "",
			level: "DEBUG",
		},
	}
}

func (j *logScanner) Scan() bool {
	for {
		if ok := j.scanner.Scan(); !ok {
			return false
		}

		// skip java stacktrace noise
		if strings.HasPrefix(j.scanner.Text(), "\tat ") || strings.HasPrefix(j.scanner.Text(), "\t... ") {
			continue
		}

		m := logRx.FindSubmatch(j.scanner.Bytes())

		if m != nil {
			j.line.msg = string(m[2])
			j.line.level = string(m[1])
		} else {
			// Some logs aren't from java (e.g. temporal) or they have a different format,
			// or the log covers multiple lines (e.g. java stack trace). In that case, use the full line
			// and reuse the level of the previous line.
			j.line.msg = j.scanner.Text()
		}
		return true
	}
}

func (j *logScanner) Err() error {
	return j.scanner.Err()
}

func getLastLogError(r io.Reader) (string, error) {
	var lines []logLine
	s := newLogScanner(r)
	for s.Scan() {
		lines = append(lines, s.line)
	}
	if s.Err() != nil {
		return "", s.Err()
	}

	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].level == "ERROR" {
			return lines[i].msg, nil
		}
	}
	return "", nil
}
