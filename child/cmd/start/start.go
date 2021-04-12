package main

import (
	"os"
	"wilfrid.me/archive-child/pkg/child"
)

// main is used to test the child server using a static configuration.
func main() {
	path, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}

	config := child.Config{
		Port:      25565,
		StorePath: path + "/store",
	}
	resolver := child.NewResolver(config)
	if err := resolver.Server().Listen(); err != nil {
		panic(err.Error())
	}
}
