package record

import (
	"context"
	"wilfrid.me/netpb/pkg/archive"
)

type Repository interface {
	Get(context.Context, int) (archive.Snapshot, error)
	Add(context.Context, archive.Snapshot) (int64, error)
	All(context.Context) ([]archive.Snapshot, error)
	GetPeer(context.Context, int) (archive.Peer, error)
	AddPeer(context.Context, archive.Peer) (int64, error)
	AllPeers(context.Context) ([]archive.Peer, error)
}
