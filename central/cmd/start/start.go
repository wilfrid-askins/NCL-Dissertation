package main

import (
	"fmt"
	"wilfrid.me/archive-central/internal"
	"wilfrid.me/netpb/pkg/netpb"
)

func main() {
	config := internal.LoadConfig()
	//resolver := internal.NewResolver(config)

	// use chi
	// create http server
	// forward to server.HTTP() or something
	// consider moving to other module

	addr := fmt.Sprintf("%s:%d", config.Server.Address, config.Server.Port)
	cl := netpb.NewClient(addr, nil)

	go func() {
		if err := cl.Connect(); err != nil {
			println(err.Error())
		}
	}()

	for !cl.Connected {}

	cl.Events.Subscribe(1, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageResponse)
		println("PageResponse")
		println(req.Found)
		println(req.ChunkNum)
		println(req.ChunkSize)
	})

	cl.Events.Subscribe(2, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		chunk := p.Message.(*netpb.PageChunk)
		println("PageChunk")
		println(chunk.Id)
		println(chunk.Chunk)
		println(chunk.Data)
	})

	if err := cl.Send(&netpb.PageRequest{
		Id: 112,
	}); err != nil {
		println(err.Error())
	}

	for {}
}
