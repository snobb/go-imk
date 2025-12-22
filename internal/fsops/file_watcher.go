package fsops

import (
	"context"
	"fmt"

	"go-imk/internal/logger"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	files []string
}

func NewFileWatcher(files []string) *FileWatcher {
	return &FileWatcher{
		files: files,
	}
}

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
				events <- &Event{
					Op:   event.Op.String(),
					Path: event.Name,
				}

			case err := <-watcher.Errors:
				logger.Shoutf("watcher error :: %s", err.Error())
				return
			}
		}
	}()

	return events, nil
}
