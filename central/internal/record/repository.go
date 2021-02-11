package record

import (
	"context"
	"wilfrid.me/netpb/pkg/archive"
)

type Repository interface {
	Get(context.Context, int) (archive.Snapshot, error)
	GetPeer(context.Context, int) (archive.Peer, error)
	Add(context.Context, archive.Snapshot) (int64, error)
}
