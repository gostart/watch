package watch

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/go-fsnotify/fsnotify"
)

var WriteFilePerm os.FileMode = 0660

type Buffer struct {
	filename string
	buffer   bytes.Buffer
	watcher  *fsnotify.Watcher
	close    chan struct{}
	mutex    sync.Mutex
}

func NewBuffer(filename string) (*Buffer, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(filename)
	if err != nil {
		return nil, err
	}
	buffer := &Buffer{
		filename: filename,
		watcher:  watcher,
		close:    make(chan struct{}),
	}
	err = buffer.readFile()
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				switch event.Op {
				case fsnotify.Create, fsnotify.Write:
					buffer.mutex.Lock()
					err := buffer.readFile()
					buffer.mutex.Unlock()
					if err != nil {
						log.Println("watch.Buffer:", err)
					}

				case fsnotify.Remove, fsnotify.Rename:
					buffer.mutex.Lock()
					buffer.buffer.Reset()
					buffer.mutex.Unlock()
				}

			case err := <-watcher.Errors:
				log.Println("watch.Buffer:", err)

			case <-buffer.close:
				return
			}
		}
	}()

	return buffer, nil
}

func (buffer *Buffer) readFile() error {
	data, err := ioutil.ReadFile(buffer.filename)
	if err != nil {
		return err
	}
	buffer.buffer.Reset()
	_, err = buffer.buffer.Write(data)
	return err
}

func (buffer *Buffer) Filename() string {
	return buffer.filename
}

func (buffer *Buffer) Bytes() []byte {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	return buffer.buffer.Bytes()
}

func (buffer *Buffer) SetBytes(data []byte) error {
	// Triggers a file read to buffer.buffer, could be optimized
	return ioutil.WriteFile(buffer.filename, data, WriteFilePerm)
}

func (buffer *Buffer) String() string {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	return buffer.buffer.String()
}

func (buffer *Buffer) SetString(str string) error {
	// Triggers a file read to buffer.buffer, could be optimized
	return ioutil.WriteFile(buffer.filename, []byte(str), WriteFilePerm)
}

func (buffer *Buffer) Close() error {
	buffer.mutex.Lock()
	defer buffer.mutex.Unlock()

	if buffer.watcher == nil {
		return nil
	}
	buffer.close <- struct{}{}
	err := buffer.watcher.Close()
	buffer.watcher = nil
	return err
}
