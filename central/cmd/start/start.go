package main

import (
	"fmt"
	"net/http"
	"wilfrid.me/archive-central/internal"
)

func main() {
	config := internal.LoadConfig()
	resolver := internal.NewResolver(config)

	ws := resolver.WebServer()
	port := fmt.Sprintf(":%d", config.Server.Port)
	err := http.ListenAndServe(port, ws.GetRouter())
	if err != nil {
		resolver.Logger().Errorw("webserver failed to listen", "msg", err.Error())
	}
}
