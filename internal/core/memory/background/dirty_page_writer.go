package background

import "context"

type DirtyPageWriter interface {
	StartBackgroundWriter() (context.CancelFunc, error)
	StartCheckPointer() (context.CancelFunc, error)
}
