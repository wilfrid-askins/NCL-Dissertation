package record

import (
	"database/sql"
	"wilfrid.me/archive-central/internal"
)

type (
 Repository struct {
	 db *sql.DB
 }

 Filter struct {
	 internal.Snapshot
 }
)

func NewRepository(db *sql.DB) *Repository{
	return &Repository{
		db: db,
	}
}

func (re *Repository) Select(f Filter) ([]internal.Snapshot, error) {
	return nil, nil
}

func (re *Repository) Add(r internal.Snapshot) (int64, error) {
	res, err := re.db.Exec("INSERT INTO pages VALUES (%s, %s)", r.URL, r.Time)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
