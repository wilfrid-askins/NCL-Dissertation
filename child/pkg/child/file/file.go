// Package file is used to store/retrieve page files.
package file

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// Repository interacts with the child server file system.
type Repository struct {
	storePath string
}

// NewRepository returns a new instance of the file repository.
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

// Get returns the page file with the given ID.
func (r *Repository) Get(id int) ([]byte, error) {
	// decide where data was stored
	path := fmt.Sprintf("%s%d.html", r.storePath, id)
	// read the snapshot data from file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading file")
	}
	return data, nil
}

// Store will save the given data for the given page ID.
func (r *Repository) Store(id int, data []byte) error {
	// decide a the file for the data
	path := fmt.Sprintf("%s%d.html", r.storePath, id)
	// write the data to file
	return ioutil.WriteFile(path, data, os.ModePerm)
}
