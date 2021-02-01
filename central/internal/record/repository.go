package record

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"time"
	"wilfrid.me/netpb/pkg/archive"
)

type (
	Repository struct {
		db    *sql.DB
		files *packr.Box
	}

	Filter struct {
		archive.Snapshot
	}
)

func NewRepository(db *sql.DB) (*Repository, error) {
	re := &Repository{
		db:    db,
		files: packr.New("Config Box", "./config"),
	}
	if err := re.create(); err != nil {
		return nil, err
	}
	return re, nil
}

func (re *Repository) create() error {
	ctx, cfn := context.WithTimeout(context.Background(), time.Second*10)
	defer cfn()
	snapshots, err := re.files.FindString("snapshots.sql")
	if err != nil {
		return errors.Wrap(err, "finding snapshots table file")
	}
	if _, err := re.db.ExecContext(ctx, snapshots); err != nil {
		return errors.Wrap(err, "executing snapshots table init")
	}

	peers, err := re.files.FindString("peers.sql")
	if err != nil {
		return errors.Wrap(err, "finding peers table file")
	}
	if _, err := re.db.ExecContext(ctx, peers); err != nil {
		return errors.Wrap(err, "executing peers table init")
	}

	return nil
}

func (re *Repository) Get(ctx context.Context, id int) (archive.Snapshot, error) {
	row := re.db.QueryRowContext(ctx, "SELECT `id`, `sum`, `url`,`time`,`origin` FROM snapshots WHERE `id`=?;", id)
	s := archive.Snapshot{}
	if err := row.Scan(&s.ID, &s.Sum, &s.URL, &s.Time, &s.Origin); err != nil {
		return archive.Snapshot{}, err
	}
	return s, nil
}

func (re *Repository) GetLast(ctx context.Context, num int) ([]archive.Snapshot, error) {
	query := fmt.Sprintf("SELECT `id`, `sum`, `url`,`time`,`origin` FROM snapshots ORDER BY `time` DESC LIMIT %d;", num)
	rows, err := re.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "querying rows")
	}
	var results []archive.Snapshot

	for rows.Next() {
		s := archive.Snapshot{}
		if err := rows.Scan(&s.ID, &s.Sum, &s.URL, &s.Time, &s.Origin); err != nil {
			return nil, errors.Wrap(err, "scanning row")
		}
		results = append(results, s)
	}

	return results, nil
}

func (re *Repository) GetPeers(ctx context.Context) ([]archive.Peer, error) {
	rows, err := re.db.QueryContext(ctx, "SELECT `id`, `address` FROM peers;")
	if err != nil {
		return nil, errors.Wrap(err, "querying rows")
	}
	var results []archive.Peer

	for rows.Next() {
		s := archive.Peer{}
		if err := rows.Scan(&s.ID, &s.Address); err != nil {
			return nil, errors.Wrap(err, "scanning row")
		}
		results = append(results, s)
	}

	return results, nil
}

func (re *Repository) GetPeer(ctx context.Context, id int) (archive.Peer, error) {
	row := re.db.QueryRowContext(ctx, "SELECT `id`, `address` FROM peers WHERE `id`=?;", id)
	s := archive.Peer{}
	if err := row.Scan(&s.ID, &s.Address); err != nil {
		return archive.Peer{}, err
	}
	return s, nil
}

func (re *Repository) Add(ctx context.Context, r archive.Snapshot) (int64, error) {
	res, err := re.db.ExecContext(ctx, "INSERT INTO snapshots(`sum`,`url`,`time`,`origin`) VALUES (?,?,?,?);", r.Sum, r.URL, r.Time, r.Origin)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
