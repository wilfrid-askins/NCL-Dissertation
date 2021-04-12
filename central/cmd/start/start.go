package main

import (
	"fmt"
	"net/http"

	"wilfrid.me/archive-central/pkg/central"
)

func main() {
	config := central.LoadConfig()
	resolver := central.NewResolver(config)

	resolver.Logger().Info("Starting...")

	ws := resolver.WebServer()
	port := fmt.Sprintf(":%d", config.Server.Port)
	err := http.ListenAndServe(port, ws.GetRouter())
	if err != nil {
		resolver.Logger().Errorw("webserver failed to listen", "msg", err.Error())
	}
}
