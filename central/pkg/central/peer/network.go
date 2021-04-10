package peer

import (
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
	sctx, cfn := context.WithTimeout(ctx, time.Second*2)
	defer cfn()
	cached, err := nw.cache.Get(sctx, key).Bytes()
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

	data, err, _ := nw.requestGroup.Do(fmt.Sprintf("network-get-%d", id), func() (interface{}, error) {
		data, err := nw.getWithFallback(ctx, snap, peers)
		return data, err
	})
	if err != nil {
		return nil, errors.Wrap(err, "messaging peers")
	}
	bdata := (data).([]byte)

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
	respData, err, _ := nw.requestGroup.Do(fmt.Sprintf("http-get-%s", url), func() (interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return 0, errors.Wrap(err, "downloading url")
		}
		return ioutil.ReadAll(resp.Body)
	})
	if err != nil {
		return 0, errors.Wrap(err, "reading url data")
	}
	data := respData.([]byte)

	checkSum, err := getHash(data)
	if err != nil {
		return 0, errors.Wrap(err, "calculating hash")
	}

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
			if err := nw.sendToMirrors(nctx, int(recID), peerID, data); err != nil {
				nw.logger.Errorw("sending data to mirrors", "id", recID, "err", err.Error())
			}
		}()
	}

	con, err := netpb.Connect(ctx, peer.Address, nw.logger)
	if err != nil {
		return 0, errors.Wrap(err, "connecting to peer")
	}
	defer con.Close()

	if err := netpb.SendPage(recID, data, con.Write); err != nil {
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

func (nw *Network) getWithFallback(ctx context.Context, snap archive.Snapshot, peers []archive.Peer) ([]byte, error) {
	for _, peer := range peers {
		data, err := nw.getFromPeer(ctx, snap, peer.Address)
		if err != nil {
			nw.logger.Warnw("couldn't get from peer", "addr", peer.Address, "msg", err.Error())
		}

		return data, nil
	}
	// TODO: get from all peers at once

	return nil, errors.New("none available")
}

func (nw *Network) getFromPeer(ctx context.Context, snap archive.Snapshot, address string) ([]byte, error) {
	nw.logger.Infow("get-file from peer", "id", snap.ID, "addr", address)
	var notFound bool
	var result []byte

	con, err := netpb.Connect(ctx, address, nw.logger)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to peer")
	}
	defer con.Close()

	if err := con.Write(&netpb.PageRequest{
		Id: int64(snap.ID),
	}); err != nil {
		return nil, errors.Wrap(err, "sending page request")
	}

loop:
	for e := range con.Rec {
		switch e.Type {
		case netpb.EventPacket:
			p := e.Packet
			switch p.(type) {
			case *netpb.PageTransfer:
				nw.logger.Infow("get-file transfer start", "id", snap.ID, "addr", address)
				trans := p.(*netpb.PageTransfer)
				if !trans.Found {
					notFound = true
					con.Close()
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
				con.Close()
				break loop
			default:
				nw.logger.Error("unknown packet type received")
			}
		case netpb.EventError:
			nw.logger.Errorw("connection returned error", "err", e.Error.Error())
		}
	}
	if notFound {
		return nil, errors.New("not found")
	}

	checkSum, err := getHash(result)
	if err != nil {
		return nil, errors.Wrap(err, "calculating hash")
	}
	if snap.Sum != checkSum {
		return nil, errors.Errorf("wrong hash, want: %s, got: %s", snap.Sum, checkSum)
	}

	nw.logger.Infow("get-file transfer end", "id", snap.ID, "addr", address)
	return result, nil
}

func getHash(data []byte) (string, error) {
	hash := sha512.New512_256()
	if _, err := hash.Write(data); err != nil {
		return "", errors.Wrap(err, "writing hash")
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
