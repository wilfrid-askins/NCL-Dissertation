package main

import "wilfrid.me/archive-central/internal"

func main() {
	config := internal.LoadConfig()
	resolver := internal.NewResolver(config)
	// use chi
	// create http server
	// forward to server.HTTP() or something
	// consider moving to other module

	// create web socket listener
	// listen
}
