package local

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// 2024-09-10 20:16:24 WARN i.m.s.r.u.Loggers$Slf4JLogger(warn):299 - [273....
var javaLogRx = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \x1b\[(?:1;)?\d+m(?P<level>[A-Z]+)\x1b\[m (?P<msg>\S+ - .*)`)

type javaLogLine struct {
	msg   string
	level string
}

type javaLogScanner struct {
	scanner *bufio.Scanner
	line    javaLogLine
}

func newJavaLogScanner(r io.Reader) *javaLogScanner {
	return &javaLogScanner{
		scanner: bufio.NewScanner(r),
		line: javaLogLine{
			msg:   "",
			level: "DEBUG",
		},
	}
}

func (j *javaLogScanner) Scan() bool {
	for {
		if ok := j.scanner.Scan(); !ok {
			return false
		}

		// skip java stacktrace noise
		if strings.HasPrefix(j.scanner.Text(), "\tat ") || strings.HasPrefix(j.scanner.Text(), "\t... ") {
			continue
		}

		m := javaLogRx.FindSubmatch(j.scanner.Bytes())

		if m != nil {
			j.line.msg = string(m[2])
			j.line.level = string(m[1])
		} else {
			j.line.msg = j.scanner.Text()
		}
		return true
	}
}

func (j *javaLogScanner) Err() error {
	return j.scanner.Err()
}

func getAllJavaLogLines(r io.Reader) ([]javaLogLine, error) {
	lines := []javaLogLine{}
	s := newJavaLogScanner(r)
	for s.Scan() {
		lines = append(lines, s.line)
	}
	return lines, s.Err()
}

func getLastJavaLogError(r io.Reader) (string, error) {
	lines, err := getAllJavaLogLines(r)
	if err != nil {
		return "", err
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].level == "ERROR" {
			return lines[i].msg, nil
		}
	}
	return "", nil
}
