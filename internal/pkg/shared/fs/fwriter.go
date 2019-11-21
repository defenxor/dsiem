package fs

import (
	"errors"
	"os"
	"path"
	"sync"
	"time"

	"github.com/enriquebris/goconcurrentqueue"
)

// FileWriter allows concurrent writes to a file
type FileWriter struct {
	filePath string
	sync.Mutex
	handle *os.File
	q      *goconcurrentqueue.FixedFIFO
	chDone chan struct{}
}

// Init setup the FileWriter
func (fw *FileWriter) Init(filePath string, queueLength int) (err error) {
	dir := path.Dir(filePath)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	fw.Lock()
	if fw.q == nil {
		fw.q = goconcurrentqueue.NewFixedFIFO(queueLength)
		fw.chDone = make(chan struct{})
		go fw.writeListener(fw.chDone)
	}
	fw.Unlock()
	err = fw.setFile(filePath)
	return
}

// SetFile set the file target
func (fw *FileWriter) setFile(filePath string) (err error) {
	fw.Lock()
	defer fw.Unlock()
	if fw.handle != nil {
		fw.handle.Close()
	}
	fw.filePath = filePath
	fw.handle, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	return
}

// EnqueueWrite writes string to the target file
func (fw *FileWriter) EnqueueWrite(data string) (err error) {
	if fw.q == nil || fw.filePath == "" || fw.handle == nil {
		err = errors.New("queue is uninitialized")
	} else {
		err = fw.q.Enqueue(data)
	}
	return
}

func (fw *FileWriter) writeListener(chDone chan struct{}) {
	for {
		res, err := fw.q.DequeueOrWaitForNextElement()
		select {
		case <-chDone:
			return
		default:
		}
		if err == nil {
			fw.Lock()
			fw.handle.SetDeadline(time.Now().Add(60 * time.Second))
			fw.handle.WriteString(res.(string) + "\n")
			fw.Unlock()
		}
	}
}

// Stop ends the file writer
func (fw *FileWriter) Stop() (err error) {
	fw.Lock()
	fw.chDone <- struct{}{}
	if fw.handle != nil {
		fw.handle.Close()
	}
	fw.handle = nil
	fw.q.Enqueue("done")
	fw.q.Lock() // will cause dequeue to return error on new data
	fw.Unlock()
	return
}
