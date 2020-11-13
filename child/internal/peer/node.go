package peer

import (
	"bytes"
	"compress/gzip"
	"go.uber.org/zap"
	"wilfrid.me/archive-child/internal/file"
	"wilfrid.me/netpb/pkg/archive"
	"wilfrid.me/netpb/pkg/netpb"
)

type Node struct {
	events *netpb.Emitter
	files *file.Repository
	transfers *netpb.Transfer
	logger *zap.SugaredLogger
}

func Subscribe(events *netpb.Emitter, files *file.Repository, logger *zap.SugaredLogger) *Node {
	node := &Node{
		events: events,
		files: files,
		transfers: netpb.NewTransfer(logger),
		logger: logger,
	}
	node.subscribe()
	return node
}

func (no *Node) subscribe() {
	no.events.Subscribe(0, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageRequest)

		data, err := no.files.Get(archive.Snapshot{
			ID: int(req.Id),
		})
		if err != nil {
			no.logger.Errorw("failed to read file", "msg", err.Error())
			return
		}

		var comp bytes.Buffer
		zw := gzip.NewWriter(&comp)

		if _, err := zw.Write(data); err != nil {
			no.logger.Errorw("failed to compress file", "msg", err.Error())
			return
		}
		if err := zw.Close(); err != nil {
			no.logger.Errorw("failed to close compression", "msg", err.Error())
			return
		}

		// read file with that Id
		// split into chunks
		// send page response packet
		// send chunks in page chunk packets

		sendData := comp.Bytes()
		chunkSize := int64(len(sendData))
		chunkNum := int64(1)
		totalSize := chunkSize

		if err := netpb.Write(p.Conn, &netpb.PageTransfer{
			Id: req.Id,
			Found:     true,
			ChunkSize: chunkSize,
			ChunkNum:  chunkNum,
			TotalSize: totalSize,
		}); err != nil {
			no.logger.Errorw("failed to send page response", "msg", err.Error())
		}

		if err := netpb.Write(p.Conn, &netpb.PageChunk{
			Id:    req.Id,
			Chunk: 0,
			Data:  sendData,
		}); err != nil {
			no.logger.Errorw("failed to send page response", "msg", err.Error())
		}
	})

	no.events.Subscribe(1, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageTransfer)
		no.transfers.New(req)
	})

	no.events.Subscribe(2, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		chunk := p.Message.(*netpb.PageChunk)
		done, result := no.transfers.Push(chunk)
		if done {
			println(string(result))
		}
	})
}