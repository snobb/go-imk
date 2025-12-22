package fsops

//go:generate moq -rm -fmt goimports -out walker_mock.go . Walker

type Walker interface {
	Walk(path string) ([]string, error)
}

type WalkerFunc func(path string) ([]string, error)

func (w WalkerFunc) Walk(path string) ([]string, error) {
	return w(path)
}
