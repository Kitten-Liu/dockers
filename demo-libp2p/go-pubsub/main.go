package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

// DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
const DiscoveryServiceTag = "pubsub-chat-example"
const httpPort = 8080

func main() {
	ctx := context.Background()

	// create a new libp2p Host that listens on a random TCP port
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	)

	if err != nil {
		panic(err)
	}

	// create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, h)

	if err != nil {
		panic(err)
	}

	addrInfo := &peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	_, err = peer.AddrInfoToP2pAddrs(addrInfo)

	if err != nil {
		panic(err)
	}

	// 自身可以作为中继节点
	r, err := relay.New(h)

	if err != nil {
		panic(err)
	}

	// setup local mDNS discovery
	ms := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h})

	if err := ms.Start(); err != nil {
		panic(err)
	}

	node := &Node{
		Ctx:    ctx,
		Host:   h,
		Mdns:   ms,
		Relay:  r,
		Pubsub: ps,
	}

	go func() {
		ginServer(httpPort)
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")

	if err := node.Close(); err != nil {
		panic(err)
	}
}

func ginServer(port int) {
	r := gin.Default()

	r.GET("/message/:msg", func(c *gin.Context) {
		req := struct {
			Msg string `uri:"msg"`
		}{}

		if err := c.ShouldBindUri(&req); err != nil {
			c.JSON(400, err.Error())
			return
		}

		c.JSON(200, req.Msg)
	})

	r.GET("/connect/*multiaddr", func(c *gin.Context) {
		req := struct {
			Addr string `uri:"multiaddr"`
		}{}

		if err := c.ShouldBindUri(&req); err != nil {
			c.JSON(400, err.Error())
			return
		}

		addr := strings.Trim(req.Addr, "/")
		msg := fmt.Sprintf("/connect /%s", addr)

		c.JSON(200, msg)
	})

	r.GET("/relay/*multiaddr", func(c *gin.Context) {
		req := struct {
			Addr string `uri:"multiaddr"`
		}{}

		if err := c.ShouldBindUri(&req); err != nil {
			c.JSON(400, err.Error())
			return
		}

		addr := strings.Trim(req.Addr, "/")
		msg := fmt.Sprintf("/relay /%s", addr)

		c.JSON(200, msg)
	})

	r.Run(fmt.Sprintf(":%d", port))
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID)
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID, err)
	}
}
