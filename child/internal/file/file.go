package file

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"wilfrid.me/netpb/pkg/archive"
)

type Repository struct {
	storePath string
}

func NewRepository(storePath string) (*Repository, error) {
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		if err := os.Mkdir(storePath, os.ModePerm); err != nil {
			return nil, errors.Wrap(err, "creating directory")
		}
	}
	return &Repository{
		storePath: storePath,
	}, nil
}

func (r *Repository) Get(id int) ([]byte, error) {
	// check cache for snapshot data
	// read snapshot data from file
	// return data
	path := fmt.Sprintf("%s%d.html", r.storePath, id)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading file")
	}
	return data, nil
}

func (r *Repository) Store(id int, data []byte) error {
	// create file of s
	path := fmt.Sprintf("%s%d.html", r.storePath, id)
	// write data to file
	return ioutil.WriteFile(path, data, os.ModePerm)
}

func fileName(s archive.Snapshot) string {
	return fmt.Sprintf("%d", s.ID)
}
