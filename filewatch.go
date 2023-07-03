package filewatch

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type file struct {
	Path    string
	Size    int64
	ModTime time.Time
	SHA256  string
}

type Options struct {
	// Interval is the duration to wait between polling.
	Interval time.Duration

	// Include are file patterns that should be watched.
	// Any directories/files that do not match will be ignored.
	Include []string

	// Exclude are file patterns that should be ignored.
	// Any directories/files that match will be ignored.
	Exclude []string
}

type Watcher struct {
	C   <-chan struct{}
	Err <-chan error

	path string
	opts *Options

	files map[string]file
	c     chan struct{}
	err   chan error
	tick  *time.Ticker
}

// New returns a new Watcher.
func New(path string, opts Options) *Watcher {
	c := make(chan struct{})
	err := make(chan error)

	w := &Watcher{
		C:    c,
		Err:  err,
		path: path,
		opts: &opts,
		c:    c,
		err:  err,
	}

	if w.opts.Interval == 0 {
		w.opts.Interval = 2 * time.Second
	} else if w.opts.Interval <= 0 {
		panic(errors.New("non-positive interval for Watcher"))
	}

	go w.start()

	return w
}

// Stop stops the Watcher. The underlying channels will be closed.
func (w *Watcher) Stop() {
	if w.tick == nil {
		return
	}

	w.tick.Stop()
	w.tick = nil

	close(w.c)
	close(w.err)
}

func (w *Watcher) start() {
	w.tick = time.NewTicker(w.opts.Interval)

	for {
		select {
		case _, ok := <-w.tick.C:
			if !ok {
				return
			}

			if err := w.walk(); err != nil {
				select {
				case w.err <- err:
				default:
				}
			}
		}
	}
}

func (w *Watcher) walk() error {
	files := make(map[string]file, len(w.files))

	err := filepath.Walk(w.path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore directories.
		if info != nil && info.IsDir() {
			return nil
		}

		// Check if the file should be processed.
		if len(w.opts.Include) > 0 {
			matched, err := pathMatches(w.opts.Include, path)
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}

		if len(w.opts.Exclude) > 0 {
			matched, err := pathMatches(w.opts.Exclude, path)
			if err != nil {
				return err
			}
			if matched {
				return nil
			}
		}

		key := file{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		f, err := os.Open(key.Path)
		if err != nil {
			return err
		}
		defer f.Close()

		var (
			h       = sha256.New()
			scanner = bufio.NewScanner(f)
		)

		for scanner.Scan() {
			if _, err := h.Write(scanner.Bytes()); err != nil {
				return err
			}
		}

		key.SHA256 = hex.EncodeToString(h.Sum(nil))

		files[path] = key

		return nil
	})
	if err != nil {
		return err
	}

	if w.hasChange(files) {
		select {
		case w.c <- struct{}{}:
		default:
		}
	}

	w.files = files // Set to latest _after_ checking if we had changes.

	return nil
}

// hasChange compares the new set of files with the old.
func (w *Watcher) hasChange(files map[string]file) bool {
	if w.files == nil {
		return false // Don't reload on the initial check.
	}

	if len(files) != len(w.files) {
		return true
	}

	// Check for new files.
	for k := range files {
		existing, ok := w.files[k]
		if !ok {
			return true
		}
		if files[k].SHA256 != existing.SHA256 {
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

// pathMatches checks if path matches any of the patterns.
func pathMatches(patterns []string, path string) (bool, error) {
	for _, p := range patterns {
		matched, err := filepath.Match(p, path)
		if err != nil {
			return false, err
		}

		if matched {
			return true, nil
		}
	}

	return false, nil
}
