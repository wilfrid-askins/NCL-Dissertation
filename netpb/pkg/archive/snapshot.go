package archive

import "time"

type Snapshot struct {
ID int
Sum string
URL string
Time time.Time
Origin string
// Secondary storage peers decided by algorithm
}

type Peer struct {
	ID int
	Address string
}
