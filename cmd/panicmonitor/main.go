package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/lujjjh/panicmonitor"

	"github.com/spf13/viper"
)

var (
	config     string
	executable string
	args       []string
	ready      = make(chan struct{})
	done       = make(chan struct{})
	cmd        *exec.Cmd
)

func printHelp() {
	fmt.Println("usage:")
	fmt.Println("  panicmonitor <config> <executable> [ ...args ]")
	os.Exit(1)
}

func parseArgs(osArgs []string) {
	if len(osArgs) < 3 {
		printHelp()
		os.Exit(1)
	}
	config = osArgs[1]
	executable = osArgs[2]
	args = osArgs[3:]
}

func readInConfig() {
	viper.SetConfigFile(config)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}
}

func runAndMonitor() {
	defer close(done)

	var err error
	messageChan := make(chan []byte)
	cmd, err = panicmonitor.Run(executable, args, messageChan)
	if err != nil {
		log.Fatalln(err)
	}
	close(ready)

	message, ok := <-messageChan
	if ok {
		panicmonitor.Report(message, &panicmonitor.ReportOptions{
			RecordFile: viper.GetString("report.record_file"),
			Throttle:   viper.GetDuration("report.throttle"),

			DingTalk: viper.GetString("dingtalk.webhook"),
		})
	}

	err = cmd.Wait()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitStatus := 1
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			exitStatus = status.ExitStatus()
		}
		os.Exit(exitStatus)
	} else if err != nil {
		log.Fatalln(err)
	}
}

func propagateSignals() {
	sigChan := make(chan os.Signal)

	signal.Notify(sigChan)
	for sig := range sigChan {
		cmd.Process.Signal(sig)
	}
}

func main() {
	parseArgs(os.Args)

	readInConfig()

	go runAndMonitor()

	<-ready
	propagateSignals()

	<-done
}
