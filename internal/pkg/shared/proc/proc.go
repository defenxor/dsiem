package proc

import (
	"os"
	"syscall"
)

// GetProcID get the current process ID
func GetProcID() int {
	return os.Getpid()
}

// StopProcess sends interrupt signal to PID
func StopProcess(pid int) (err error) {
	// ignore error since it always return nil except on Windows
	proc, _ := os.FindProcess(pid)
	// actually check the process existence
	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		return
	}
	err = proc.Signal(os.Interrupt)
	return
}
