package file

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"wilfrid.me/netpb/pkg/archive"
)

type Repository struct {
	directory string
}

func NewRepository() (*Repository, error) {
	path, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "finding working directory")
	}

	directory := path + "/store/"
	if err := os.Mkdir(directory, os.ModeDir); err != nil {
		return nil, errors.Wrap(err, "creating store directory")
	}

	return &Repository{
		directory: directory,
	}, nil
}

func Get(s archive.Snapshot) ([]byte, error) {
	// check cache for snapshot data
	// read snapshot data from file
	// return data
	return nil, nil
}

func Store(s archive.Snapshot, data []byte) error {
	// create file of s
	// write data
	return nil
}

func fileName(s archive.Snapshot) string {
	return fmt.Sprintf("%d", s.ID)
}
