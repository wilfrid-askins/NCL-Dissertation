// Package peer provides functionality for interacting with the files stored by a group of machines.
package peer

import (
	"context"
)

// Group describes any peer group with Get/Store functions.
type Group interface {
	// Get returns the file of the page with the specified id.
	Get(ctx context.Context, pageID int) ([]byte, error)
	// Store will query a url and store its content in the specified peer.
	Store(ctx context.Context, peerID int, url string) (int, error)
}
