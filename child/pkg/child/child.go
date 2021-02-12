package child

import (
	"wilfrid.me/archive-child/pkg/child/peer"
	"wilfrid.me/netpb/pkg/netpb"
)

func NewListener(config Config) *netpb.Listener {
	resolver := NewResolver(config)
	_ = peer.New(resolver.Listener().Events, resolver.Files(), resolver.Logger())
	return resolver.Listener()
}
