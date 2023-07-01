package filewatch

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"time"
)

type file struct {
	Path    string
	Size    int64
	ModTime time.Time
}

type Watcher struct {
	C <-chan struct{}

	path  string
	files map[file]struct{}
	c     chan struct{}
	tick  *time.Ticker
}

func New(path string) *Watcher {
	c := make(chan struct{}, 1)

	w := &Watcher{
		C:    c,
		path: path,
		c:    c,
	}

	go w.start()

	return w
}

func (w *Watcher) Stop() {
	if w.tick == nil {
		return
	}

	w.tick.Stop()
	w.tick = nil

	close(w.c)
}

func (w *Watcher) start() {
	w.tick = time.NewTicker(250 * time.Millisecond)

	for {
		select {
		case _, ok := <-w.tick.C:
			if !ok {
				return
			}

			if err := w.walk(); err != nil {
				fmt.Println("err", err)
			}
		}
	}
}

func (w *Watcher) walk() error {
	files := make(map[file]struct{})

	err := filepath.Walk(w.path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info != nil && info.IsDir() && w.path == path {
			return nil
		}

		files[file{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}] = struct{}{}

		return nil
	})
	if err != nil {
		return err
	}

	if w.hasChange(files) {
		w.c <- struct{}{}
	}

	w.files = files // Set to latest _after_ checking if we had changes.

	return nil
}

func (w *Watcher) hasChange(files map[file]struct{}) bool {
	if w.files == nil {
		return false // Don't reload on the initial check.
	}

	if len(files) != len(w.files) {
		return true
	}

	// Check for new files.
	for k := range files {
		if _, exists := w.files[k]; !exists {
			return true
		}
	}

	// Check for deleted files.
	for k := range w.files {
		if _, exists := files[k]; !exists {
			return true
		}
	}

	return false
}
