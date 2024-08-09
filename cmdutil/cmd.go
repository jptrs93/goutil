package cmdutil

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

func InitStdPipes(cmd *exec.Cmd) (io.ReadCloser, io.ReadCloser, io.WriteCloser, func(), error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("creating stdout pipe %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return nil, nil, nil, nil, fmt.Errorf("creating stderr pipe %v", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = stderr.Close()
		_ = stdout.Close()
		return nil, nil, nil, nil, fmt.Errorf("creating stdin pipe %v", err)
	}
	closeFunc := func() {
		_ = stdout.Close()
		_ = stderr.Close()
		_ = stdin.Close()
	}
	return stdout, stderr, stdin, closeFunc, nil
}

func FindProcessUsingPort(port int) (*int, error) {
	// Use lsof to find the process listening on the specified port
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%v", port), "-t")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute lsof: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return nil, nil
	}

	pid, err := strconv.Atoi(outputStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PID: %v", err)
	}

	return &pid, nil
}
