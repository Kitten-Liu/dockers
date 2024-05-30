package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	libp2ppubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

type Node struct {
	Ctx    context.Context
	Host   libp2phost.Host
	Mdns   mdns.Service
	Relay  *relay.Relay
	Pubsub *libp2ppubsub.PubSub
}

func (n *Node) Close() error {
	if err := n.Mdns.Close(); err != nil {
		return err
	}

	if err := n.Relay.Close(); err != nil {
		return err
	}

	if err := n.Host.Close(); err != nil {
		return err
	}

	return nil
}

func (n *Node) RelayTo(destAddress string) error {
	relayInfo, err := peer.AddrInfoFromString(destAddress)

	if err != nil {
		LogError("AddrInfoFromString error: %s, address: %s", err.Error(), destAddress)
		return err
	}

	ctx, cancel := context.WithTimeout(n.Ctx, 15*time.Second)
	defer cancel()

	n.Host.Network().(*swarm.Swarm).Backoff().Clear(relayInfo.ID)
	n.Host.Peerstore().AddAddrs(relayInfo.ID, relayInfo.Addrs, peerstore.PermanentAddrTTL)

	if err := n.Host.Connect(ctx, *relayInfo); err != nil {
		LogError("Relay to node fail: %s, info: %v", err.Error(), relayInfo)
		return err
	}

	rCtx, _ := context.WithTimeout(n.Ctx, 15*time.Second)

	// 注册到中继节点，允许中继节点代理请求
	if _, err := client.Reserve(rCtx, n.Host, *relayInfo); err != nil {
		LogError("Reserve fail: %s, info: %v", err.Error(), relayInfo)
		return err
	}

	for _, raddr := range relayInfo.Addrs {
		a := fmt.Sprintf("%s/p2p/%s/p2p-circuit/p2p/%s", raddr.String(), relayInfo.ID, n.Host.ID())
		LogInfo("Your address via relay is %s", url.QueryEscape(a))
	}

	return nil
}
