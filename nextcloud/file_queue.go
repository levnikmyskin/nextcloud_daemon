package nextcloud

import (
	"sync"
	"errors"
)

// Simple thread-safe stack to enqueue file and later download them
// https://stackoverflow.com/questions/28541609/looking-for-reasonable-stack-implementation-in-golang
type FileStack struct {
	lock sync.Mutex
	stack []string
}

func NewFileStack() *FileStack {
	return &FileStack{sync.Mutex{}, make([]string, 0)}
}

func (f *FileStack) Push(file string) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.stack = append(f.stack, file)
}

func (f *FileStack) Pop() (string, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	l := len(f.stack)
	if l == 0 {
		return "", errors.New("empty stack")
	}

	file := f.stack[l-1]
	f.stack = f.stack[:l-1]

	return file, nil
}

func (f *FileStack) Len() int {
	return len(f.stack)
}
