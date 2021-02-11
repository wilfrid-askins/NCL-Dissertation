package main

import (
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
	listener := child.NewListener(config)
	if err := listener.Listen(); err != nil {
		panic(err.Error())
	}
}
