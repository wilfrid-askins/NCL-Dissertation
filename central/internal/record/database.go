package record

import (
	"context"
	"database/sql"
	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"time"
	"wilfrid.me/netpb/pkg/archive"
)

type Database struct {
	db    *sql.DB
	files *packr.Box
}

func NewDatabase(db *sql.DB) (*Database, error) {
	re := &Database{
		db:    db,
		files: packr.New("Config Box", "./config"),
	}
	if err := re.create(); err != nil {
		return nil, err
	}
	return re, nil
}

func (db *Database) create() error {
	ctx, cfn := context.WithTimeout(context.Background(), time.Second*10)
	defer cfn()
	snapshots, err := db.files.FindString("snapshots.sql")
	if err != nil {
		return errors.Wrap(err, "finding snapshots table file")
	}
	if _, err := db.db.ExecContext(ctx, snapshots); err != nil {
		return errors.Wrap(err, "executing snapshots table init")
	}

	peers, err := db.files.FindString("peers.sql")
	if err != nil {
		return errors.Wrap(err, "finding peers table file")
	}
	if _, err := db.db.ExecContext(ctx, peers); err != nil {
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

func (db *Database) GetPeer(ctx context.Context, id int) (archive.Peer, error) {
	row := db.db.QueryRowContext(ctx, "SELECT `id`, `address` FROM peers WHERE `id`=?;", id)
	s := archive.Peer{}
	if err := row.Scan(&s.ID, &s.Address); err != nil {
		return archive.Peer{}, err
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
