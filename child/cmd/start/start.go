package main

import (
	"context"
	"os"
	"wilfrid.me/archive-child/pkg/child"
)

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
	if err := resolver.Server().Listen(context.Background()); err != nil {
		panic(err.Error())
	}
}
