package internal

import "time"

type Snapshot struct {
	ID int
	URL string
	Time time.Time
	// Peer ID?
	// Secondary storage peers decided by algorithm
}
