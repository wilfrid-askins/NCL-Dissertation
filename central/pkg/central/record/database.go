// Package records provides functionality for storing the metadata of peers, snapshots and mirrors.
package record

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"wilfrid.me/archive-central/pkg/central/core"
)

//go:embed tables/snapshots.sql
var snapshots string

//go:embed tables/peers.sql
var peers string

//go:embed tables/mirrors.sql
var mirrors string

var tables = []string{
	peers, snapshots, mirrors,
}

// Database is used to query a MySQL database for records.
type Database struct {
	db *sql.DB
}

// NewDatabase returns a records repository using the supplied SQL connection.
func NewDatabase(db *sql.DB) (*Database, error) {
	re := &Database{db: db}
	if err := re.create(); err != nil {
		return nil, err
	}
	return re, nil
}

func (db *Database) create() error {
	ctx, cfn := context.WithTimeout(context.Background(), time.Second*10)
	defer cfn()

	for _, table := range tables {
		_, err := db.db.ExecContext(ctx, table)
		if err != nil {
			return errors.Wrap(err, "executing table init")
		}
	}
	return nil
}

// Get returns a snapshot record with the given ID.
func (db *Database) Get(ctx context.Context, id int) (core.Snapshot, error) {
	row := db.db.QueryRowContext(ctx, "SELECT s.id, s.sum, s.url, s.time, p.id, p.address FROM snapshots s INNER JOIN peers p ON p.id=s.peerID WHERE s.id=?;", id)
	s := core.Snapshot{}
	if err := row.Scan(&s.ID, &s.Sum, &s.URL, &s.Time, &s.Peer.ID, &s.Peer.Address); err != nil {
		return core.Snapshot{}, err
	}
	return s, nil
}

// Add will store a snapshot record.
func (db *Database) Add(ctx context.Context, r core.Snapshot) (int64, error) {
	res, err := db.db.ExecContext(ctx, "INSERT INTO snapshots(`sum`,`url`,`time`,`peerID`) VALUES (?,?,?,?);", r.Sum, r.URL, r.Time, r.Peer.ID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// All will return all snapshot records in the database.
func (db *Database) All(ctx context.Context) ([]core.Snapshot, error) {
	rows, err := db.db.QueryContext(ctx, "SELECT `id`, `sum`, `url`,`time`,`peerID` FROM snapshots;")
	if err != nil {
		return nil, errors.Wrap(err, "querying all snapshots")
	}
	defer rows.Close()
	var results []core.Snapshot
	for rows.Next() {
		s := core.Snapshot{}
		if err := rows.Scan(&s.ID, &s.Sum, &s.URL, &s.Time, &s.Peer.ID); err != nil {
			return nil, errors.Wrap(err, "scanning snapshot")
		}
		results = append(results, s)
	}
	return results, nil
}

// GetPeer will return the peer with the given ID.
func (db *Database) GetPeer(ctx context.Context, id int) (core.Peer, error) {
	row := db.db.QueryRowContext(ctx, "SELECT `id`, `address` FROM peers WHERE `id`=?;", id)
	s := core.Peer{}
	if err := row.Scan(&s.ID, &s.Address); err != nil {
		return core.Peer{}, err
	}
	return s, nil
}

// AddPeer will add a peer record to the database.
func (db *Database) AddPeer(ctx context.Context, p core.Peer) (int64, error) {
	res, err := db.db.ExecContext(ctx, "INSERT INTO peers(`address`) VALUES (?);", p.Address)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetPeers will return all peers in the database with an optional limit.
// If a limit is supplied, all peers will be given back in a random order.
func (db *Database) GetPeers(ctx context.Context, limit int) ([]core.Peer, error) {
	query := "SELECT `id`, `address` FROM peers"
	if limit > 0 {
		query += fmt.Sprintf(" ORDER BY RAND() LIMIT %d", limit)
	}
	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "querying all peers")
	}
	defer rows.Close()
	var results []core.Peer
	for rows.Next() {
		p := core.Peer{}
		if err := rows.Scan(&p.ID, &p.Address); err != nil {
			return nil, errors.Wrap(err, "scanning peer")
		}
		results = append(results, p)
	}
	return results, nil
}

// GetMirrors will return all the mirrors of a given snapshot ID.
func (db *Database) GetMirrors(ctx context.Context, snapID int) ([]core.Peer, error) {
	rows, err := db.db.QueryContext(ctx, "SELECT p.id, p.address FROM mirrors m INNER JOIN peers p ON m.peerID=p.id WHERE m.snapshotID=?;", snapID)
	if err != nil {
		return nil, errors.Wrap(err, "querying mirrors")
	}
	defer rows.Close()
	var results []core.Peer
	for rows.Next() {
		p := core.Peer{}
		if err := rows.Scan(&p.ID, &p.Address); err != nil {
			return nil, errors.Wrap(err, "scanning peer")
		}
		results = append(results, p)
	}
	return results, nil
}

// AddMirror will add a record of a mirror for the given peer and snapshot.
func (db *Database) AddMirror(ctx context.Context, peerID, snapID int) (int64, error) {
	res, err := db.db.ExecContext(ctx, "INSERT INTO mirrors(`peerID`, `snapshotID`) VALUES (?, ?);", peerID, snapID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
