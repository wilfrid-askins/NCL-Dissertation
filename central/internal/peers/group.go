package peers

import (
	"bytes"
	"fmt"
	"sync"
	"wilfrid.me/netpb/pkg/netpb"
)

type (
 transfer struct {
	ID int64
	ChunkNum, ChunkSize int64
	Data *bytes.Buffer
}

 Group struct {
	 transfers map[int64]transfer
	 transferMx sync.RWMutex
}
)

func NewGroup() *Group {
	return &Group{
		transfers: make(map[int64]transfer),
		transferMx: sync.RWMutex{},
	}
}

func (gr *Group) Get(id int) ([]byte, error){
	// TODO determine client to
	finished := false
	var result []byte

	address := "127.0.0.1"
	port := 25565

	addr := fmt.Sprintf("%s:%d", address, port)
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

		if !req.Found {
			println("NOT FOUND")
			return
		}

		data := make([]byte, req.TotalSize)
		buffer := bytes.NewBuffer(data)

		page := transfer{
			ID:        req.Id,
			ChunkNum:  req.ChunkNum,
			ChunkSize: req.ChunkSize,
			Data:    buffer,
		}

		gr.transferMx.Lock()
		defer gr.transferMx.Unlock()
		gr.transfers[req.Id] = page
	})

	cl.Events.Subscribe(2, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		chunk := p.Message.(*netpb.PageChunk)
		println("PageChunk")

		gr.transferMx.Lock()
		defer gr.transferMx.Unlock()
		page := gr.transfers[chunk.Id]
		page.Data.Write(chunk.Data)

		if chunk.Chunk == page.ChunkNum-1 {
			println("PAGE DONE")
			result = page.Data.Bytes()
			finished = true
		}
	})

	if err := cl.Send(&netpb.PageRequest{
		Id: int64(id),
	}); err != nil {
		println(err.Error())
	}

	for !finished {}
	return result, nil
}