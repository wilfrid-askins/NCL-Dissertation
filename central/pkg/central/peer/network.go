package peer

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
	"wilfrid.me/archive-central/pkg/central/record"
	"wilfrid.me/netpb/pkg/archive"
	"wilfrid.me/netpb/pkg/netpb"
)

const (
	cacheTime      = time.Hour
	searchFactor   = 4
	mirrorTime     = time.Minute * 2
	firstCacheTime = time.Minute * 10
)

type (
	Network struct {
		records      record.Repository
		firstCache   map[int]*cached
		firstCacheMx sync.RWMutex
		cache        *redis.Client
		transfers    *netpb.Transfer
		requestGroup singleflight.Group
		logger       *zap.SugaredLogger
		mirrors      int
	}

	cached struct {
		data []byte
		age  time.Time
	}
)

func NewNetwork(records record.Repository, cache *redis.Client, mirrors int, logger *zap.SugaredLogger) *Network {
	return &Network{
		records:      records,
		firstCache:   make(map[int]*cached),
		firstCacheMx: sync.RWMutex{},
		cache:        cache,
		transfers:    netpb.NewTransfer(logger),
		mirrors:      mirrors,
		logger:       logger,
	}
}

/*
func (nw *Network) Get(ctx context.Context, id int) ([]byte, error) {
	//nw.firstCacheMx.RLock()
	//defer nw.firstCacheMx.RUnlock()
	//if cached, pres := nw.firstCache[id]; pres {
	//	nw.logger.Info("first cache hit")
	//	cached.age = time.Now()
	//	return cached.data, nil
	//}

	data, err, _ := nw.requestGroup.Do(fmt.Sprintf("page-%d", id), func() (interface{}, error) {
		data, err := nw.getWork(ctx, id)
		return data, err
	})
	bdata := (data).([]byte)

	//nw.firstCacheMx.Lock()
	//defer nw.firstCacheMx.Unlock()
	//nw.firstCache[id] = &cached{
	//	data: bdata,
	//	age:  time.Now(),
	//}
	return bdata, err
}
*/

func (nw *Network) Get(ctx context.Context, id int) ([]byte, error) {
	// check cache
	key := fmt.Sprintf("snapshot-%d", id)
	cached, err := nw.cache.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		nw.logger.Errorw("failed to query cache", "msg", err.Error())
	}
	if err != redis.Nil {
		return cached, nil
	}

	nw.logger.Infow("getting record", "id", id)
	// find peer
	snap, err := nw.records.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting snapshot record")
	}
	mirrors, err := nw.records.GetMirrors(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting mirror records")
	}
	// make array of peers to query
	var peers []archive.Peer
	peers = append(peers, snap.Peer)
	peers = append(peers, mirrors...)

	nw.logger.Infow("messaging peers", "id", id)

	//data, err, _ := nw.requestGroup.Do(fmt.Sprintf("network-%d", id), func() (interface{}, error) {
	//	data, err := nw.getWithFallback(ctx, id, peers)
	//	return data, err
	//})
	data, err := nw.getWithFallback(ctx, id, peers)
	if err != nil {
		return nil, errors.Wrap(err, "getting from peers")
	}
	bdata := data // (data).([]byte)

	if err := nw.cache.Set(ctx, key, bdata, cacheTime).Err(); err != nil {
		nw.logger.Errorw("failed to set cache", "msg", err.Error())
	}
	return bdata, nil
}

func (nw *Network) Store(ctx context.Context, peerID int, url string) (int, error) {
	// find peer to store in
	peer, err := nw.records.GetPeer(ctx, peerID)
	if err != nil {
		return 0, errors.Wrap(err, "getting peer record")
	}

	// download url to memory
	resp, err := http.Get(url)
	if err != nil {
		return 0, errors.Wrap(err, "downloading url")
	}
	pageData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.Wrap(err, "reading url data")
	}

	// compress page data
	var comp bytes.Buffer
	zw := gzip.NewWriter(&comp)
	if _, err := zw.Write(pageData); err != nil {
		return 0, errors.Wrap(err, "compressing page data")
	}
	if err := zw.Close(); err != nil {
		return 0, errors.Wrap(err, "closing compression writer")
	}
	data := comp.Bytes()

	// calculate hash of page data
	hash := sha512.New512_256()
	if _, err := hash.Write(data); err != nil {
		return 0, errors.Wrap(err, "writing hash")
	}
	checkSum := fmt.Sprintf("%x", hash.Sum(nil))

	// add page record
	recID, err := nw.records.Add(ctx, archive.Snapshot{
		Sum:  checkSum,
		URL:  url,
		Time: time.Now(),
		Peer: archive.Peer{
			ID: peerID,
		},
	})
	if err != nil {
		return 0, errors.Wrap(err, "creating new page record")
	}

	if nw.mirrors > 0 {
		go func() {
			nctx, cfn := context.WithTimeout(context.Background(), mirrorTime)
			defer cfn()
			if err := nw.sendToMirrors(nctx, int(recID), peerID, comp.Bytes()); err != nil {
				nw.logger.Errorw("sending data to mirrors", "id", recID, "err", err.Error())
			}
		}()
	}

	con, err := netpb.Connect(ctx, peer.Address, nw.logger)
	if err != nil {
		return 0, errors.Wrap(err, "connecting to peer")
	}
	defer con.Close()

	if err := netpb.SendPage(recID, comp.Bytes(), con.Write); err != nil {
		return 0, errors.Wrap(err, "sending page data")
	}
	return int(recID), nil
}

func (nw *Network) sendToMirrors(ctx context.Context, recID, exID int, data []byte) error {
	peers, err := nw.records.GetPeers(ctx, nw.mirrors*searchFactor)
	if err != nil {
		return errors.Wrap(err, "getting list of peers")
	}
	needed := nw.mirrors
	for needed > 0 && len(peers) > 0 {
		peer := peers[0]
		peers = peers[1:]
		if peer.ID == exID {
			continue
		}

		con, err := netpb.Connect(ctx, peer.Address, nw.logger)
		if err != nil {
			nw.logger.Errorw("connecting to mirror peer", "msg", err.Error())
			continue
		}
		if err := netpb.SendPage(int64(recID), data, con.Write); err != nil {
			nw.logger.Errorw("sending page", "msg", err.Error())
			continue
		}

		if _, err := nw.records.AddMirror(ctx, peer.ID, recID); err != nil {
			nw.logger.Errorw("adding mirror record", "msg", err.Error())
			continue
		}
		needed--
		con.Close()
	}
	if needed > 0 {
		return errors.New("failed to find enough available peers")
	}
	return nil
}

func (nw *Network) getWithFallback(ctx context.Context, id int, peers []archive.Peer) ([]byte, error) {
	for _, peer := range peers {
		data, err := nw.getFromPeer(ctx, id, peer.Address)
		if err != nil {
			nw.logger.Warnw("getting page from peer", "msg", err.Error())
			continue
		}
		return data, nil
	}
	return nil, errors.New("none available")
}

func (nw *Network) getFromPeer(ctx context.Context, id int, address string) ([]byte, error) {
	nw.logger.Infow("messaging peer", "id", id, "addr", address)
	var notFound bool
	var result []byte

	con, err := netpb.Connect(ctx, address, nw.logger)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to peer")
	}
	defer con.Close()

	if err := con.Write(&netpb.PageRequest{
		Id: int64(id),
	}); err != nil {
		return nil, errors.Wrap(err, "sending page request")
	}

loop:
	for {
		select {
		case p := <-con.Rec:
			switch p.(type) {
			case *netpb.PageTransfer:
				nw.logger.Infow("transfer from", "id", id, "addr", address)
				trans := p.(*netpb.PageTransfer)
				if !trans.Found {
					notFound = true
					break loop
				}
				nw.transfers.New(trans)
			case *netpb.PageChunk:
				chunk := p.(*netpb.PageChunk)
				done, data := nw.transfers.Push(chunk)
				if !done {
					continue
				}
				result = data
				break loop
			default:
				nw.logger.Error("unknown packet type received")
			}
		case err := <-con.Errs:
			nw.logger.Errorw("connection returned error", "err", err.Error())
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if notFound {
		return nil, errors.New("not found")
	}

	// TODO test checksum here

	nw.logger.Infow("transfer finished from", "id", id, "addr", address)
	return result, nil
}
