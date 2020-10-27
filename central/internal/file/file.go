package file

import (
	"fmt"
	"wilfrid.me/archive-central/internal"
)

type Repository struct {

}

func NewRepository() Repository {
	return Repository{}
}

func Get(s internal.Snapshot) ([]byte, error) {
	// check cache for snapshot data
	// read snapshot data from file
	// return data
}

func Store(s internal.Snapshot, data []byte) error {
	// create file of s
	// write data
}

func fileName(s internal.Snapshot) string {
	return fmt.Sprintf("%d", s.ID)
}
