package record

import (
	"context"
	"errors"
	"sort"
	"wilfrid.me/netpb/pkg/archive"
)

type FakeDatabase struct {
	Snapshots []archive.Snapshot
	Peers     []archive.Peer
}

func (db *FakeDatabase) Get(_ context.Context, id int) (archive.Snapshot, error) {
	idx := sort.Search(len(db.Snapshots), func(i int) bool {
		return db.Snapshots[i].ID == id
	})
	if idx == -1 {
		return archive.Snapshot{}, errors.New("snapshot not found")
	}
	return db.Snapshots[idx], nil
}

func (db *FakeDatabase) GetPeer(_ context.Context, id int) (archive.Peer, error) {
	idx := sort.Search(len(db.Peers), func(i int) bool {
		return db.Peers[i].ID == id
	})
	if idx == -1 {
		return archive.Peer{}, errors.New("peer not found")
	}
	return db.Peers[idx], nil
}

func (db *FakeDatabase) Add(_ context.Context, snapshot archive.Snapshot) (int64, error) {
	var id int64
	if len(db.Snapshots) > 0 {
		id = int64(db.Snapshots[len(db.Snapshots)-1].ID + 1)
	}
	snapshot.ID = int(id)
	db.Snapshots = append(db.Snapshots, snapshot)
	return id, nil
}
