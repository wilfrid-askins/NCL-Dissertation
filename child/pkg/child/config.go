// Package child implements the functionality of the archive child server.
package child

// Config is used to describe the port and file path for the child server to operate on.
type Config struct {
	Port      int
	StorePath string
}
