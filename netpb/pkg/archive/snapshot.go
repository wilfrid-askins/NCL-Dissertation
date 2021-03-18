package archive

import "time"

type Snapshot struct {
	ID   int       `json:"id"`
	Sum  string    `json:"sum"`
	URL  string    `json:"url"`
	Time time.Time `json:"time"`
	Peer Peer      `json:"peer"`
}

type Peer struct {
	ID      int    `json:"id"`
	Address string `json:"address"`
}
