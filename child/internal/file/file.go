package file

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"wilfrid.me/netpb/pkg/archive"
)

type Repository struct {
	basePath string
	storePath string
}

func NewRepository() (*Repository, error) {
	path, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "finding working directory")
	}

	basePath := path + "/archive/"
	storePath := basePath + "/store/"
	paths := []string{
		basePath,
		storePath,
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.Mkdir(path, os.ModePerm); err != nil {
				return nil, errors.Wrap(err, "creating directory")
			}
		}
	}

	return &Repository{
		basePath: basePath,
		storePath: storePath,
	}, nil
}

func (r *Repository) Get(s archive.Snapshot) ([]byte, error) {
	// check cache for snapshot data
	// read snapshot data from file
	// return data
	path := fmt.Sprintf("%s%d.html", r.storePath, s.ID)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading file")
	}
	return data, nil
}

func Store(s archive.Snapshot, data []byte) error {
	// create file of s
	// write data
	return nil
}

func fileName(s archive.Snapshot) string {
	return fmt.Sprintf("%d", s.ID)
}
