package archive

import "time"

type Snapshot struct {
	ID     int       `json:"id"`
	Sum    string    `json:"sum"`
	URL    string    `json:"url"`
	Time   time.Time `json:"time"`
	Origin string    `json:"origin"`
	// Secondary storage peers decided by algorithm
}

type Peer struct {
	ID      int    `json:"id"`
	Address string `json:"address"`
}
