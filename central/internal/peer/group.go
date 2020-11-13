package peer

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
	"wilfrid.me/archive-central/internal/record"
	"wilfrid.me/netpb/pkg/archive"
	"wilfrid.me/netpb/pkg/netpb"
)

type (
	Group struct {
		records   *record.Repository
		logger    *zap.SugaredLogger
		transfers *netpb.Transfer
	}
)

func NewGroup(records *record.Repository, logger *zap.SugaredLogger) *Group {
	return &Group{
		records:   records,
		logger:    logger,
		transfers: netpb.NewTransfer(logger),
	}
}

func (gr *Group) Get(id int) ([]byte, error) {
	snap, err := gr.records.Get(id)
	if err != nil {
		return nil, errors.Wrap(err, "getting snapshot record")
	}

	return gr.GetFromPeer(id, snap.Origin)
	// TODO try mirrors if not found
}

func (gr *Group) Store(peerID int, url string) error {
	// determine peer
	// download web page
	// send page transfer
	peer, err := gr.records.GetPeer(peerID)
	if err != nil {
		return errors.Wrap(err, "getting peer record")
	}

	// download url to memory
	// get checksum of bytes
	// store record in db
	// get id of record
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "downloading url")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "reading url data")
	}
	recID, err := gr.records.Add(archive.Snapshot{
		Sum:    "abc123",
		URL:    url,
		Time:   time.Now(),
		Origin: peer.Address,
	})

	cl := netpb.NewClient(peer.Address, gr.logger)

	go func() {
		if err := cl.Connect(); err != nil {
			gr.logger.Errorw("connecting to peer", "err", err)
		}
	}()

	for !cl.Connected {
	}

	if err := cl.Send(&netpb.PageTransfer{
		Id:        recID,
		ChunkSize: len(data),
		ChunkNum:  1,
		TotalSize: len(data),
	}); err != nil {
		return errors.Wrap(err, "page transfer msg")
	}
	if err := cl.Send(&netpb.PageChunk{
		Id:    recID,
		Chunk: 0,
		Data:  data,
	}); err != nil {
		return errors.Wrap(err, "page chunk msg")
	}

	return nil
}

func (gr *Group) GetFromPeer(id int, address string) ([]byte, error) {
	var done bool
	var result []byte
	cl := netpb.NewClient(address, gr.logger)

	go func() {
		if err := cl.Connect(); err != nil {
			gr.logger.Errorw("connecting to peer", "err", err)
		}
	}()

	for !cl.Connected {
	}

	cl.Events.Subscribe(1, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageTransfer)
		if !req.Found {
			// TODO not found
			return
		}
		gr.transfers.New(req)
	})

	cl.Events.Subscribe(2, func(i interface{}) {
		p := i.(netpb.PacketEvent)
		chunk := p.Message.(*netpb.PageChunk)
		done, result = gr.transfers.Push(chunk)
	})

	if err := cl.Send(&netpb.PageRequest{
		Id: int64(id),
	}); err != nil {
		gr.logger.Errorw("sending page request", "err", err)
	}

	for !done {
	}

	return result, nil
}
