package main

import (
	"wilfrid.me/archive-child/internal"
	"wilfrid.me/netpb/pkg/netpb"
)

func main() {
	config := internal.LoadConfig()
	resolver := internal.NewResolver(config)

	events := resolver.Listener().Events
	events.Subscribe(0, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageRequest)
		println(req.Id)

		// read file with that Id
		// split into chunks
		// send page response packet
		// send chunks in page chunk packets

		if err := netpb.Write(p.Conn, &netpb.PageResponse{
			Found:     true,
			ChunkSize: 128,
			ChunkNum:  12,
		}); err != nil {
			resolver.Logger().Errorw("failed to send page response", "msg", err.Error())
		}
	})

	if err := resolver.Listener().Listen(); err != nil {
		resolver.Logger().Fatalw("starting listening", "msg", err.Error())
	}
}
