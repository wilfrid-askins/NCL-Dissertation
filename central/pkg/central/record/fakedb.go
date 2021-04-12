package record

import (
	"context"
	"errors"
	"sort"

	"wilfrid.me/archive-central/pkg/central/core"
)

type FakeDatabase struct {
	Snapshots []core.Snapshot
	Peers     []core.Peer
}

func (db *FakeDatabase) Get(_ context.Context, id int) (core.Snapshot, error) {
	idx := sort.Search(len(db.Snapshots), func(i int) bool {
		return db.Snapshots[i].ID == id
	})
	if idx == -1 {
		return core.Snapshot{}, errors.New("snapshot not found")
	}
	return db.Snapshots[idx], nil
}

func (db *FakeDatabase) Add(_ context.Context, snapshot core.Snapshot) (int64, error) {
	var id int64
	if len(db.Snapshots) > 0 {
		id = int64(db.Snapshots[len(db.Snapshots)-1].ID + 1)
	}
	snapshot.ID = int(id)
	db.Snapshots = append(db.Snapshots, snapshot)
	return id, nil
}

func (db *FakeDatabase) All(_ context.Context) ([]core.Snapshot, error) {
	return db.Snapshots, nil
}

func (db *FakeDatabase) GetPeer(_ context.Context, id int) (core.Peer, error) {
	idx := sort.Search(len(db.Peers), func(i int) bool {
		return db.Peers[i].ID == id
	})
	if idx == -1 {
		return core.Peer{}, errors.New("peer not found")
	}
	return db.Peers[idx], nil
}

func (db *FakeDatabase) AddPeer(_ context.Context, peer core.Peer) (int64, error) {
	peer.ID = len(db.Peers)
	db.Peers = append(db.Peers, peer)
	return int64(peer.ID), nil
}

func (db *FakeDatabase) GetPeers(_ context.Context, limit int) ([]core.Peer, error) {
	return db.Peers[0:min(limit, len(db.Peers))], nil
}

func (db *FakeDatabase) GetMirrors(_ context.Context, _ int) ([]core.Peer, error) {
	return nil, nil
}

func (db *FakeDatabase) AddMirror(_ context.Context, _, _ int) (int64, error) {
	return 0, errors.New("not yet implemented")
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
