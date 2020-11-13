package main

import (
	"fmt"
	"net/http"
	"wilfrid.me/archive-central/internal"
)

func main() {
	config := internal.LoadConfig()
	resolver := internal.NewResolver(config)

	//_, err := resolver.Records().Add(archive.Snapshot{
	//	Sum:    "abcdefgh",
	//	URL:    "https://google.com",
	//	Time:   time.Now(),
	//	Origin: "localhost:25565",
	//})
	//if err != nil {
	//	panic(err)
	//}
	//
	//s, err := resolver.Records().Get(1)
	//if err != nil {
	//	panic(err)
	//}
	//println(s.ID)
	//println(s.Origin)
	//println(s.URL)

	ws := resolver.WebServer()
	port := fmt.Sprintf(":%d", config.Server.Port)
	err := http.ListenAndServe(port, ws.GetRouter())
	if err != nil {
		resolver.Logger().Errorw("webserver failed to listen", "msg", err.Error())
	}
}
