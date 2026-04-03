package utils

import (
	"errors"
	"io"
	"os"
	"runtime"
	"sync"
)

var (
	consoleWriteMu sync.Mutex

	stdoutConsoleWriter io.Writer = newConsoleOutputWriter(
		func() io.Writer { return os.Stdout },
		runtime.GOOS == "windows",
	)

	stderrConsoleWriter io.Writer = newConsoleOutputWriter(
		func() io.Writer { return os.Stderr },
		runtime.GOOS == "windows",
	)
)

// StdoutWriter returns a synchronized writer for terminal stdout output.
func StdoutWriter() io.Writer {
	return stdoutConsoleWriter
}

// StderrWriter returns a synchronized writer for terminal stderr output.
func StderrWriter() io.Writer {
	return stderrConsoleWriter
}

type consoleOutputWriter struct {
	target             func() io.Writer
	normalizeWindowsLF bool
}

func newConsoleOutputWriter(target func() io.Writer, normalizeWindowsLF bool) *consoleOutputWriter {
	return &consoleOutputWriter{
		target:             target,
		normalizeWindowsLF: normalizeWindowsLF,
	}
}

func (w *consoleOutputWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	toWrite := p
	if w.normalizeWindowsLF {
		toWrite = normalizeConsoleNewlinesWindows(p)
	}

	consoleWriteMu.Lock()
	defer consoleWriteMu.Unlock()

	target := w.target()
	if target == nil {
		return 0, errors.New("console output target is nil")
	}

	n, err := target.Write(toWrite)
	if err != nil {
		return normalizedBytesToSourceBytes(p, n), err
	}
	if n != len(toWrite) {
		return normalizedBytesToSourceBytes(p, n), io.ErrShortWrite
	}

	return len(p), nil
}

func normalizeConsoleNewlinesWindows(src []byte) []byte {
	needsNormalization := false
	var prev byte
	for i, b := range src {
		if b == '\n' && (i == 0 || prev != '\r') {
			needsNormalization = true
			break
		}
		prev = b
	}
	if !needsNormalization {
		return src
	}

	extra := 0
	prev = 0
	for i, b := range src {
		if b == '\n' && (i == 0 || prev != '\r') {
			extra++
		}
		prev = b
	}

	dst := make([]byte, 0, len(src)+extra)
	prev = 0
	for i, b := range src {
		if b == '\n' && (i == 0 || prev != '\r') {
			dst = append(dst, '\r', '\n')
		} else {
			dst = append(dst, b)
		}
		prev = b
	}

	return dst
}

func normalizedBytesToSourceBytes(src []byte, normalizedBytes int) int {
	if normalizedBytes <= 0 {
		return 0
	}

	consumedNormalized := 0
	var prev byte
	for i, b := range src {
		width := 1
		if b == '\n' && (i == 0 || prev != '\r') {
			width = 2
		}

		if consumedNormalized+width > normalizedBytes {
			return i
		}

		consumedNormalized += width
		prev = b
	}

	return len(src)
}
