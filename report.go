package panicmonitor

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var hostname, _ = os.Hostname()

// ReportOptions contains some options that Report uses.
type ReportOptions struct {
	RecordFile string
	Throttle   time.Duration

	DingTalk string
}

func shouldReport(recordFile string, throttle time.Duration) bool {
	data, err := ioutil.ReadFile(recordFile)
	if err != nil {
		return true
	}

	timestamp, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return true
	}

	lastReportedAt := time.Unix(timestamp, 0)
	return time.Since(lastReportedAt) >= throttle
}

func reportPanicViaDingTalk(webhook string, message []byte) {
	maxLines := 20

	summary := ""
	lines := bytes.SplitN(message, []byte("\n"), maxLines+1)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		summary = "..."
	}
	for i := range lines {
		lines[i] = append([]byte("    "), lines[i]...)
	}

	title := "Panic from " + hostname
	summary = string(bytes.Join(lines, []byte("\n"))) + summary
	body, _ := json.Marshal(map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": title,
			"text":  "### " + title + "\n\n" + summary,
		},
	})
	http.Post(webhook, "application/json", bytes.NewReader(body))
}

func writeRecord(recordFile string) {
	timestamp := time.Now().Unix()
	ioutil.WriteFile(recordFile, []byte(strconv.FormatInt(timestamp, 10)), 0644)
}

// Report reports a panic message.
func Report(message []byte, options *ReportOptions) {
	recordFile := options.RecordFile
	if recordFile == "" {
		recordFile = "/tmp/panicmonitor"
	}

	throttle := options.Throttle
	minThrottle := 1 * time.Minute
	if throttle < minThrottle {
		throttle = minThrottle
	}

	if !shouldReport(recordFile, throttle) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	if options.DingTalk != "" {
		wg.Add(1)
		go func() {
			reportPanicViaDingTalk(options.DingTalk, message)
			wg.Done()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return
	case <-done:
		writeRecord(recordFile)
	}
}
