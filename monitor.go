package panicmonitor

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"time"
)

var panicHeaders = [][]byte{
	[]byte("panic:"),
	[]byte("fatal error:"),
}

func indexPanicHeader(s []byte) int {
	for _, h := range panicHeaders {
		if p := bytes.Index(s, h); p >= 0 {
			return p
		}
	}
	return -1
}

func tracePanicLikeStuffs(r io.Reader, messageChan chan<- []byte) error {
	var buf [2048]byte
	var panicBuf bytes.Buffer
	var panicHeaderDetectedAt *time.Time

	for {
		var pos int

		n, err := r.Read(buf[:])
		if n == 0 {
			goto LoopTailer
		}

		// Panic header has already been detected and
		// treat what is read as a part of panic message.
		if panicHeaderDetectedAt != nil {
			panicBuf.Write(buf[:n])
			goto LoopTailer
		}

		// Let's seek for the panic header.
		pos = indexPanicHeader(buf[:n])
		if pos == -1 {
			goto LoopTailer
		}

		// Panic header is detected.
		panicHeaderDetectedAt = new(time.Time)
		*panicHeaderDetectedAt = time.Now()
		panicBuf.Write(buf[pos:n])

	LoopTailer:
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// Clear panicHeaderDetectedAt if the panic-like message is
		// detected over 300ms before now which means it should not
		// be a unrecoverable panic.
		if panicHeaderDetectedAt != nil && time.Since(*panicHeaderDetectedAt) > 300*time.Millisecond {
			panicHeaderDetectedAt = nil
			panicBuf.Reset()
		}
	}

	if panicHeaderDetectedAt == nil {
		close(messageChan)
		return nil
	}

	messageChan <- panicBuf.Bytes()

	return nil
}

// Run runs the executable with given args
// and monitors the stderr for things like panics.
func Run(executable string, args []string, messageChan chan<- []byte) (*exec.Cmd, error) {
	cmd := exec.Command(executable, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	r := io.TeeReader(stderr, os.Stderr)
	go tracePanicLikeStuffs(r, messageChan)

	return cmd, nil
}
