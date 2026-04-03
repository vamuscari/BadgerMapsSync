package utils

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestNormalizeConsoleNewlinesWindows(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "lf_only",
			in:   "line one\nline two\n",
			want: "line one\r\nline two\r\n",
		},
		{
			name: "existing_crlf_not_doubled",
			in:   "line one\r\nline two\n",
			want: "line one\r\nline two\r\n",
		},
		{
			name: "carriage_return_sequences_preserved",
			in:   "spin\rframe\rdone",
			want: "spin\rframe\rdone",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := string(normalizeConsoleNewlinesWindows([]byte(tc.in)))
			if got != tc.want {
				t.Fatalf("unexpected normalization result:\nwant=%q\ngot =%q", tc.want, got)
			}
		})
	}
}

func TestConsoleOutputWriterSerializesConcurrentWrites(t *testing.T) {
	var dst bytes.Buffer
	writer := newConsoleOutputWriter(func() io.Writer {
		return &dst
	}, false)

	start := make(chan struct{})
	var wg sync.WaitGroup

	writeMany := func(chunk string) {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			if _, err := writer.Write([]byte(chunk)); err != nil {
				t.Errorf("write failed: %v", err)
				return
			}
		}
	}

	// Use long chunks so any interleaving is easy to detect.
	chunkA := strings.Repeat("A", 96) + "\n"
	chunkB := strings.Repeat("B", 96) + "\n"

	wg.Add(2)
	go writeMany(chunkA)
	go writeMany(chunkB)

	close(start)
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(dst.String()), "\n")
	if len(lines) != 400 {
		t.Fatalf("expected 400 lines, got %d", len(lines))
	}

	for i, line := range lines {
		if line != strings.TrimSpace(chunkA) && line != strings.TrimSpace(chunkB) {
			t.Fatalf("line %d appears interleaved or corrupted: %q", i, line)
		}
	}
}
