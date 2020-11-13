package main

import (
	"wilfrid.me/archive-child/internal"
	"wilfrid.me/archive-child/internal/peer"
)

func main() {
	config := internal.LoadConfig()
	resolver := internal.NewResolver(config)

	_ = peer.Subscribe(resolver.Listener().Events, resolver.Files(), resolver.Logger())

	if err := resolver.Listener().Listen(); err != nil {
		resolver.Logger().Fatalw("starting listening", "msg", err.Error())
	}
}
