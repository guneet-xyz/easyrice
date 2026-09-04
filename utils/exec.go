package utils

import (
	"errors"
	"log/slog"
	"os/exec"
)

var ErrorFailedToRunCommand = errors.New("Failed to run command")
var ErrorBadExitCode = errors.New("Bad exit code")
var ErrorOutputNotEmpty = errors.New("Output is not empty")

func Exec(name string, args ...string) (string, error) {
	slog.Debug("Performing command", "name", name, "args", args)

	cmd := exec.Command(name, args...)

	bytes, err := cmd.CombinedOutput()
	exitCode := cmd.ProcessState.ExitCode()
	out := string(bytes)
	if err != nil {
		slog.Warn("Failed to run command", "error", err, "exitCode", exitCode, "out", out)
		return "", ErrorFailedToRunCommand
	}
	if exitCode != 0 {
		slog.Warn("Bad exit code", "exitCode", exitCode, "out", out)
		return "", ErrorBadExitCode
	}
	slog.Debug("Performed command", "out", out)
	return out, nil
}

func ExecNoOutput(name string, args ...string) error {
	slog.Debug("Performing command with no expected output", "name", name, "args", args)
	out, err := Exec(name, args...)
	if err != nil {
		return err
	}
	if len(out) > 0 {
		slog.Warn("Output is not empty")
		return ErrorOutputNotEmpty
	}
	return nil
}
