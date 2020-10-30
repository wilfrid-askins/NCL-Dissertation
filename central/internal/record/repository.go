package record

import (
	"database/sql"
	"wilfrid.me/netpb/pkg/archive"
)

type (
 Repository struct {
	 db *sql.DB
 }

 Filter struct {
	 archive.Snapshot
 }
 )

func NewRepository(db *sql.DB) (*Repository, error){
	re := &Repository{
		db: db,
	}
	if err := re.create(); err != nil {
		return nil, err
	}
	return re, nil
}

func (re *Repository) create() error {
	// check for existing tables
	// if not exists, create
	return nil
}

func (re *Repository) Select(f Filter) ([]archive.Snapshot, error) {
	return nil, nil
}

func (re *Repository) Add(r archive.Snapshot) (int64, error) {
	res, err := re.db.Exec("INSERT INTO pages VALUES (%s, %s)", r.URL, r.Time)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
