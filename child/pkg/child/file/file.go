package file

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
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
	// decide where data was stored
	path := fmt.Sprintf("%s%d.html", r.storePath, id)
	// read snapshot data from file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading file")
	}
	return data, nil
}

func (r *Repository) Store(id int, data []byte) error {
	// decide file for data
	path := fmt.Sprintf("%s%d.html", r.storePath, id)
	// write data to file
	return ioutil.WriteFile(path, data, os.ModePerm)
}
