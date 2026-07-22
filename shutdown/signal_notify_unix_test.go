//go:build !windows

package shutdown

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

const (
	notifyHelperEnvironment = "FOUNDATION_SHUTDOWN_NOTIFY_HELPER"
	notifyHelperGraceful    = "graceful"
	notifyHelperEscalated   = "escalated"
	notifyHelperReady       = "ready"
	notifyHelperCancelled   = "cancelled"
)

type notifyHelperProcess struct {
	command *exec.Cmd
	scanner *bufio.Scanner
}

func TestNotifyHandlesRealInterruptAndRestoresOperatingSystemDefault(t *testing.T) {
	if mode := os.Getenv(notifyHelperEnvironment); mode != "" {
		runNotifyHelper(t, mode)
		return
	}

	graceful := startNotifyHelper(t, notifyHelperGraceful)
	if err := graceful.command.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal(graceful interrupt) error = %v", err)
	}
	if observed := scanNotifyHelper(t, graceful); observed != notifyHelperCancelled {
		t.Fatalf("graceful helper output = %q, want %q", observed, notifyHelperCancelled)
	}
	if err := graceful.command.Wait(); err != nil {
		t.Fatalf("graceful helper Wait() error = %v", err)
	}

	escalated := startNotifyHelper(t, notifyHelperEscalated)
	if err := escalated.command.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal(first escalated interrupt) error = %v", err)
	}
	if observed := scanNotifyHelper(t, escalated); observed != notifyHelperCancelled {
		t.Fatalf("escalated helper output = %q, want %q", observed, notifyHelperCancelled)
	}
	if err := escalated.command.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Signal(second escalated interrupt) error = %v", err)
	}
	err := escalated.command.Wait()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("escalated helper Wait() error = %v, want signal ExitError", err)
	}
	status, ok := exitError.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGINT {
		t.Fatalf("escalated helper status = %v, want SIGINT termination", exitError.Sys())
	}
}

func runNotifyHelper(t *testing.T, mode string) {
	t.Helper()

	controller, err := Notify(NotifyRequest{
		Parent: context.Background(),
		Set:    SignalSetStandard,
		Policy: defaultSignalPolicy(),
	})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if _, err := fmt.Fprintln(os.Stdout, notifyHelperReady); err != nil {
		t.Fatalf("write helper readiness error = %v", err)
	}
	<-controller.Context().Done()
	<-controller.Done()
	var cause SignalCause
	if !errors.As(context.Cause(controller.Context()), &cause) || cause.Kind != SignalKindInterrupt {
		t.Fatalf("Notify() cause = %v, want interrupt SignalCause", context.Cause(controller.Context()))
	}
	if _, err := fmt.Fprintln(os.Stdout, notifyHelperCancelled); err != nil {
		t.Fatalf("write helper cancellation error = %v", err)
	}
	if mode == notifyHelperEscalated {
		select {}
	}
	if mode != notifyHelperGraceful {
		t.Fatalf("helper mode = %q, want known mode", mode)
	}
}

func startNotifyHelper(t *testing.T, mode string) notifyHelperProcess {
	t.Helper()

	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestNotifyHandlesRealInterruptAndRestoresOperatingSystemDefault$")
	command.Env = append(os.Environ(), notifyHelperEnvironment+"="+mode)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	helper := notifyHelperProcess{command: command, scanner: bufio.NewScanner(stdout)}
	if observed := scanNotifyHelper(t, helper); observed != notifyHelperReady {
		t.Fatalf("helper output = %q, want %q", observed, notifyHelperReady)
	}
	return helper
}

func scanNotifyHelper(t *testing.T, helper notifyHelperProcess) string {
	t.Helper()

	if !helper.scanner.Scan() {
		t.Fatalf("helper output ended: %v", helper.scanner.Err())
	}
	return helper.scanner.Text()
}
