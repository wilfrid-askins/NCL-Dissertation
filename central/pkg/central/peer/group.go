package peer

import (
	"context"
)

type Group interface {
	Get(context.Context, int) ([]byte, error)
	Store(context.Context, int, string) (int, error)
}
