// Package core handles the basic data types of the archive system.
package core

import "time"

// Snapshot represents a URL which was archived at a time.
type Snapshot struct {
	ID   int       `json:"id"`
	Sum  string    `json:"sum"`
	URL  string    `json:"url"`
	Time time.Time `json:"time"`
	Peer Peer      `json:"peer"`
}

// Peer represents a peer registered with the archive service.
type Peer struct {
	ID      int    `json:"id"`
	Address string `json:"address"`
}
