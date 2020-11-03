package main

import (
	"net/http"
	"wilfrid.me/archive-central/internal"
)

func main() {
	config := internal.LoadConfig()
	resolver := internal.NewResolver(config)

	// use chi
	// create http server
	// forward to server.HTTP() or something
	// consider moving to other module

	ws := resolver.WebServer()
	err := http.ListenAndServe(":3000", ws.GetRouter())
	if err != nil {
		resolver.Logger().Errorw("webserver failed to listen", "msg", err.Error())
	}

	//g := peers.NewGroup()
	//data, err := g.Get(112)
	//if err != nil {
	//	println(err.Error())
	//}
	//println(string(data))

}
