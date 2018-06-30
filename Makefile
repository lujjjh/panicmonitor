panicmonitor:
	go build -o $@ ./cmd/panicmonitor

.PHONY: install
install:
	go install ./cmd/panicmonitor
