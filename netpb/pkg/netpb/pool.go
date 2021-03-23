package netpb

import (
	"context"
	"github.com/pkg/errors"
	"sync"
)

type (
	Pool struct {
		users   map[string]occupant
		usersMx sync.Mutex
	}

	occupant struct {
		connection Connection
		users      int
	}
)

func NewPool() *Pool {
	return &Pool{
		users:   make(map[string]occupant),
		usersMx: sync.Mutex{},
	}
}

func (p *Pool) Connect(address string) (Connection, context.CancelFunc, error) {
	p.usersMx.Lock()
	defer p.usersMx.Unlock()

	occ, pres := p.users[address]
	if !pres {
		// connect
		ctx := context.Background()
		con, err := Connect(ctx, address, nil) // TODO: logger here
		if err != nil {
			return Connection{}, nil, errors.Wrap(err, "connecting to address")
		}
		occ = occupant{
			connection: con,
			users:      0,
		}
		p.users[address] = occ
	} else {
		println("cached connection")
	}
	occ.users++

	return occ.connection, func() {
		p.usersMx.Lock()
		defer p.usersMx.Unlock()
		occ.users--
		if occ.users == 0 {
			// disconnect
			occ.connection.Close()
			delete(p.users, address)
		}
	}, nil
}
