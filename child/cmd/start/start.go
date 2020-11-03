package main

import (
	"wilfrid.me/archive-child/internal"
	"wilfrid.me/netpb/pkg/archive"
	"wilfrid.me/netpb/pkg/netpb"
)

func main() {
	config := internal.LoadConfig()
	resolver := internal.NewResolver(config)

	events := resolver.Listener().Events
	events.Subscribe(0, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageRequest)

		data, err := resolver.Files().Get(archive.Snapshot{
			ID: int(req.Id),
		})
		if err != nil {
			resolver.Logger().Errorw("failed to read file", "msg", err.Error())
			return
		}

		// read file with that Id
		// split into chunks
		// send page response packet
		// send chunks in page chunk packets

		chunkSize := int64(len(data))
		chunkNum := int64(1)
		totalSize := chunkSize

		if err := netpb.Write(p.Conn, &netpb.PageResponse{
			Id: req.Id,
			Found:     true,
			ChunkSize: chunkSize,
			ChunkNum:  chunkNum,
			TotalSize: totalSize,
		}); err != nil {
			resolver.Logger().Errorw("failed to send page response", "msg", err.Error())
		}

		if err := netpb.Write(p.Conn, &netpb.PageChunk{
			Id:    req.Id,
			Chunk: 0,
			Data:  data,
		}); err != nil {
			resolver.Logger().Errorw("failed to send page response", "msg", err.Error())
		}
	})

	if err := resolver.Listener().Listen(); err != nil {
		resolver.Logger().Fatalw("starting listening", "msg", err.Error())
	}
}
