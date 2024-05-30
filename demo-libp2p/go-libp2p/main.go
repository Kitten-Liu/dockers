package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/pflag"
)

var (
	peerInit   PeerInit
	self       *Chat
	privateKey crypto.PrivKey
	// publicKey  crypto.PubKey
)

func init() {
	pflag.StringVarP(&peerInit.Host, "ip", "i", "0.0.0.0", "ipv4 address to listen")
	pflag.IntVarP(&peerInit.Port, "port", "p", 16600, "tcp port to listen")
	pflag.IntVarP(&peerInit.HttpPort, "httpport", "P", 8080, "http port that Used to receive messages sent to other peers")
	pflag.StringVarP(&peerInit.RelayAddress, "relay", "r", "", "relay node address")
}

const (
	protocol_id_chat = "/Chat/1.0.0"
)

func main() {
	var err error
	pflag.Parse()

	interfaces, err := net.Interfaces()

	if err != nil {
		panic(err)
	}

	var maddrs []multiaddr.Multiaddr

	for _, eth := range interfaces {
		addrs, err := eth.Addrs()

		if err != nil {
			LogError("device[%s].Addrs.err=%s", eth.Name, err.Error())
			continue
		}

		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil {
					addr := fmt.Sprintf("/ip4/%s/tcp/%d", ipnet.IP.String(), peerInit.Port)
					srcMultiAddr, err := multiaddr.NewMultiaddr(addr)

					if err != nil {
						panic(err)
					}

					maddrs = append(maddrs, srcMultiAddr)
				}
			}
		}
	}

	// debug local
	// maddrs = maddrs[:1]
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	privateKey, _, err = NewRSAKey(seed, 2048)

	if err != nil {
		panic(err)
	}

	// privateKeyRaw, _ := privateKey.Raw()
	// publicKeyRaw, _ := publicKey.Raw()
	// fmt.Printf("privateKey=%s\n\n", hex.EncodeToString(privateKeyRaw))
	// fmt.Printf("publicKey=%s\n\n", hex.EncodeToString(publicKeyRaw))
	// panic("adsfasf")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	options := []libp2p.Option{
		libp2p.Identity(privateKey),
		libp2p.ListenAddrs(maddrs...),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
		// libp2p.NoSecurity,
	}

	var relayInfo *peer.AddrInfo

	if len(peerInit.RelayAddress) > 0 {
		relayInfo, err = peer.AddrInfoFromString(peerInit.RelayAddress)

		if err != nil {
			LogError("invalid relay address: %s", peerInit.RelayAddress)
			panic(err)
		}

		staticRelays := []peer.AddrInfo{*relayInfo}
		options = append(options, libp2p.EnableAutoRelayWithStaticRelays(staticRelays))
	}

	node, err := libp2p.New(options...)

	if err != nil {
		panic(err)
	}

	nodeInfo := &peer.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}

	nodeAddrs, err := peer.AddrInfoToP2pAddrs(nodeInfo)

	if err != nil {
		panic(err)
	}

	// 自身可以作为中继节点
	_, err = relay.New(node)

	if err != nil {
		panic(err)
	}

	messageBox := make(chan string, 1000)
	self = &Chat{
		ID:            node.ID().String(),
		Name:          node.ID().ShortString(),
		Config:        peerInit,
		SendBox:       messageBox,
		Context:       ctx,
		Host:          node,
		Addrs:         nodeAddrs,
		AddressNotify: make(chan *Contacts, 10),
		AddressBook:   make([]*Contacts, 0, 10),
	}

	node.SetStreamHandler(protocol_id_chat, self.StreamHandler)

	self.Online()

	go func() {
		self.LetsTalk()
	}()

	go func() {
		ginServer(peerInit.HttpPort, messageBox)
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Received signal, shutting down...")

	if err := node.Close(); err != nil {
		panic(err)
	}
}

func ginServer(port int, ch chan string) {
	r := gin.Default()

	r.GET("/message/:msg", func(c *gin.Context) {
		req := struct {
			Msg string `uri:"msg"`
		}{}

		if err := c.ShouldBindUri(&req); err != nil {
			c.JSON(400, err.Error())
			return
		}

		ch <- req.Msg
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
		ch <- msg
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
		ch <- msg
		c.JSON(200, msg)
	})

	r.Run(fmt.Sprintf(":%d", port))
}
