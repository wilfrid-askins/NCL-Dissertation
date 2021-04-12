package record

import (
	"context"

	"wilfrid.me/archive-central/pkg/central/core"
)

// Repository describes the functionality to store the metadata of peers, snapshots and mirrors
type Repository interface {
	// Get returns a snapshot record with the given ID.
	Get(context.Context, int) (core.Snapshot, error)
	// Add will store a snapshot record.
	Add(context.Context, core.Snapshot) (int64, error)
	// All will return all snapshot records in the database.
	All(context.Context) ([]core.Snapshot, error)
	// GetPeer will return the peer with the given ID.
	GetPeer(context.Context, int) (core.Peer, error)
	// AddPeer will add a peer record to the database.
	AddPeer(context.Context, core.Peer) (int64, error)
	// GetPeers will return all peers in the database with an optional limit.
	GetPeers(context.Context, int) ([]core.Peer, error)
	// GetMirrors will return all the mirrors of a given snapshot ID.
	GetMirrors(context.Context, int) ([]core.Peer, error)
	// AddMirror will add a record of a mirror for the given peer and snapshot.
	AddMirror(ctx context.Context, peerID int, snapID int) (int64, error)
}
