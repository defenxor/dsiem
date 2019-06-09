package proc

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestProc(t *testing.T) {

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)

	go func() {
		<-signalChan
	}()

	myPID := os.Getpid()
	if GetProcID() != myPID {
		t.Error("GetProcID should return os.Getpid()")
	}

	err := StopProcess(31337)
	if err == nil {
		t.Error("StopProcess should return err for non-existent PID")
	}
	err = StopProcess(myPID)
	if err != nil {
		t.Error("StopProcess should not return err for this process PID. Err found: ", err)
	}

}
