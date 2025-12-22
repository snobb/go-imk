package fsops

import "context"

//go:generate moq -rm -fmt goimports -out watcher_mock.go . Watcher

type Event struct {
	Op   string
	Path string
}

type Watcher interface {
	Watch(context.Context) (chan *Event, error)
}
