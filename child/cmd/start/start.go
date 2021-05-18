package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"wilfrid.me/archive-child/pkg/child"
)

// main is used to test the child server using a static configuration.
func main() {
	path, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter port number: ")
	text, _, err := reader.ReadLine()
	if err != nil {
		panic(err.Error())
	}
	port, err := strconv.Atoi(string(text))
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Listening...")
	config := child.Config{
		Port:      port,
		StorePath: path + "/store/",
	}
	resolver := child.NewResolver(config)
	if err := resolver.Server().Listen(); err != nil {
		panic(err.Error())
	}
}
