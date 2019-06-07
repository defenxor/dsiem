package proc

import "os"

// GetProcID get the current process ID
func GetProcID() int {
	return os.Getpid()
}

// StopProcess sends interrupt signal to PID
func StopProcess(pid int) (err error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	err = proc.Signal(os.Interrupt)
	return
}
