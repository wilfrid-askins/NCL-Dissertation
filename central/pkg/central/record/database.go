package record

import (
	"context"
	"database/sql"
	_ "embed"
	"github.com/pkg/errors"
	"time"
	"wilfrid.me/netpb/pkg/archive"
)

//go:embed tables/snapshots.sql
var snapshots string

//go:embed tables/peers.sql
var peers string

type Database struct {
	db *sql.DB
}

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

	_, err := db.db.ExecContext(ctx, snapshots)
	if err != nil {
		return errors.Wrap(err, "executing snapshots table init")
	}

	_, err = db.db.ExecContext(ctx, peers)
	if err != nil {
		return errors.Wrap(err, "executing peers table init")
	}

	return nil
}

func (db *Database) Get(ctx context.Context, id int) (archive.Snapshot, error) {
	row := db.db.QueryRowContext(ctx, "SELECT `id`, `sum`, `url`,`time`,`origin` FROM snapshots WHERE `id`=?;", id)
	s := archive.Snapshot{}
	if err := row.Scan(&s.ID, &s.Sum, &s.URL, &s.Time, &s.Origin); err != nil {
		return archive.Snapshot{}, err
	}
	return s, nil
}

func (db *Database) Add(ctx context.Context, r archive.Snapshot) (int64, error) {
	res, err := db.db.ExecContext(ctx, "INSERT INTO snapshots(`sum`,`url`,`time`,`origin`) VALUES (?,?,?,?);", r.Sum, r.URL, r.Time, r.Origin)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *Database) All(ctx context.Context) ([]archive.Snapshot, error) {
	rows, err := db.db.QueryContext(ctx, "SELECT `id`, `sum`, `url`,`time`,`origin` FROM snapshots;")
	if err != nil {
		return nil, errors.Wrap(err, "querying all snapshots")
	}
	defer rows.Close()
	var results []archive.Snapshot
	for rows.Next() {
		s := archive.Snapshot{}
		if err := rows.Scan(&s.ID, &s.Sum, &s.URL, &s.Time, &s.Origin); err != nil {
			return nil, errors.Wrap(err, "scanning snapshot")
		}
		results = append(results, s)
	}
	return results, nil
}

func (db *Database) GetPeer(ctx context.Context, id int) (archive.Peer, error) {
	row := db.db.QueryRowContext(ctx, "SELECT `id`, `address` FROM peers WHERE `id`=?;", id)
	s := archive.Peer{}
	if err := row.Scan(&s.ID, &s.Address); err != nil {
		return archive.Peer{}, err
	}
	return s, nil
}

func (db *Database) AddPeer(ctx context.Context, p archive.Peer) (int64, error) {
	res, err := db.db.ExecContext(ctx, "INSERT INTO peers(`address`) VALUES (?);", p.Address)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *Database) AllPeers(ctx context.Context) ([]archive.Peer, error) {
	rows, err := db.db.QueryContext(ctx, "SELECT `id`, `address` FROM peers;")
	if err != nil {
		return nil, errors.Wrap(err, "querying all peers")
	}
	defer rows.Close()
	var results []archive.Peer
	for rows.Next() {
		p := archive.Peer{}
		if err := rows.Scan(&p.ID, &p.Address); err != nil {
			return nil, errors.Wrap(err, "scanning peer")
		}
		results = append(results, p)
	}
	return results, nil
}
