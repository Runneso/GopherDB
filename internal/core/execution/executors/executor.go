package executors

import "errors"

var ErrNotOpen = errors.New("executor is not open")

type Executor interface {
	Open() error
	Next() ([]any, error)
	Close() error
}
