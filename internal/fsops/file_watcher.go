package fsops

import (
	"context"
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"

	"github.com/snobb/go-imk/internal/logger"
)

const (
	OpCreate = "CREATE"
	OpWrite  = "WRITE"
	OpRename = "RENAME"
	OpRemove = "REMOVE"
)

type FileWatcher struct {
	files             []string
	triggerEveryEvent bool
}

// NewFileWatcher creates a new FileWatcher instance with the provided files and options.
func NewFileWatcher(files []string, triggerEveryEvent bool) *FileWatcher {
	fw := &FileWatcher{
		files:             files,
		triggerEveryEvent: triggerEveryEvent,
	}

	return fw
}

// Watch starts watching the specified files for changes and returns a channel that emits events
// when changes occur.
func (f *FileWatcher) Watch(ctx context.Context) (chan *Event, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("unable to create watcher > %w", err)
	}

	for _, file := range f.files {
		if err := watcher.Add(file); err != nil {
			return nil, fmt.Errorf("unable to watch file %s > %w", file, err)
		}
	}

	events := make(chan *Event)

	go func() {
		defer watcher.Close()
		defer close(events)

		for {
			select {
			case <-ctx.Done():
				logger.Shout("shutting down file watcher")
				return

			case event := <-watcher.Events:
				updateWatcher(watcher, &event)

				if !f.triggerEveryEvent {
					events <- &Event{Op: event.Op.String(), Path: event.Name}
				} else {
					select {
					case events <- &Event{Op: event.Op.String(), Path: event.Name}:
					default: // drop events if there is already one waiting.
					}
				}

			case err := <-watcher.Errors:
				logger.Shoutf("watcher error :: %s", err.Error())
				return
			}
		}
	}()

	return events, nil
}

func updateWatcher(watcher *fsnotify.Watcher, event *fsnotify.Event) {
	switch event.Op {
	case fsnotify.Create:
		fileInfo, err := os.Stat(event.Name)
		if err != nil || !fileInfo.IsDir() {
			return // ignore if not a folder.
		}

		if err := watcher.Add(event.Name); err != nil {
			logger.Shoutf("unable to watch file %s > %s", event.Name, err.Error())
		}

	case fsnotify.Remove, fsnotify.Rename:
		// The `fsnotify` library emits two events when a file is created:
		//
		// Event{Op: Rename, Name: "/tmp/file"}
		// Event{Op: Create, Name: "/tmp/rename", RenamedFrom: "/tmp/file"}
		//
		// So here rename returns the old name and create returns the new name, hence removing
		// here and let CREATE event handle adding new watcher.
		//
		// Since we cannot check at this point if the renamed/removed item was a folder, do best
		// effort remove and ignore errors if any.
		_ = watcher.Remove(event.Name)
	}
}
