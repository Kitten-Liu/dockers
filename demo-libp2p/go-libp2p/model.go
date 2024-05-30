package main

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	libp2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/multiformats/go-multiaddr"
)

type PeerInit struct {
	Host         string
	Port         int
	HttpPort     int
	RelayAddress string
}

type Chat struct {
	ID            string
	Name          string
	Config        PeerInit
	Context       context.Context
	Host          libp2phost.Host
	Addrs         []multiaddr.Multiaddr
	SendBox       chan string
	AddressNotify chan *Contacts
	AddressBook   []*Contacts
}

type Contacts struct {
	ID         string
	Name       string
	Context    context.Context
	Stream     network.Stream
	ReadWriter *bufio.ReadWriter
}

type Message struct {
	Time    time.Time
	Speaker *Contacts
	Content string
}

func (p *Chat) StreamHandler(s network.Stream) {
	LogInfo("New peer %s is coming", s.Conn().ID())
	p.AddContacts(s)
}

func (p *Chat) Online() {
	for _, addr := range p.Addrs {
		LogInfo("You are Online, Your p2p address is %s", url.QueryEscape(addr.String()))
	}
}

func (p *Chat) Offline() {
	LogInfo("bye")
	// disconnect
}

func (p *Chat) LetsTalk() {
	go p.Send()
	go p.Receive()

	tick := time.NewTicker(time.Minute)
	defer tick.Stop()

	select {
	case <-p.Context.Done():
		p.Offline()
		break
	case <-tick.C:
		LogInfo("%d peer online", len(p.AddressBook)+1)
	}
}

func (p *Chat) NewSession(destAddress string) error {
	maddr, err := multiaddr.NewMultiaddr(destAddress)

	if err != nil {
		return err
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)

	if err != nil {
		return err
	}

	var isViaRelay bool

	if strings.Contains(destAddress, "/p2p-circuit/") {
		info.Addrs = []multiaddr.Multiaddr{maddr}
		isViaRelay = true
	}

	LogInfo("NewSession address info: %v", info)

	p.Host.Network().(*swarm.Swarm).Backoff().Clear(info.ID)
	p.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	ctx, cancel := context.WithTimeout(p.Context, 15*time.Second)
	defer cancel()

	if err := p.Host.Connect(ctx, *info); err != nil {
		return err
	}

	var sCtx context.Context

	if isViaRelay {
		sCtx = network.WithUseTransient(p.Context, "connect via relay")
	} else {
		var sCancel context.CancelFunc
		sCtx, sCancel = context.WithTimeout(p.Context, 15*time.Second)
		defer sCancel()

	}

	s, err := p.Host.NewStream(sCtx, info.ID, protocol_id_chat)

	if err != nil {
		println("newstream error")
		return err
	}

	p.AddContacts(s)
	return nil
}

func (p *Chat) NewRelay(destAddress string) error {
	relayInfo, err := peer.AddrInfoFromString(destAddress)

	if err != nil {
		LogError("invalid relay address: %s", destAddress)
		return err
	}

	ctx, cancel := context.WithTimeout(p.Context, 15*time.Second)
	defer cancel()

	p.Host.Network().(*swarm.Swarm).Backoff().Clear(relayInfo.ID)
	p.Host.Peerstore().AddAddrs(relayInfo.ID, relayInfo.Addrs, peerstore.PermanentAddrTTL)

	if err := p.Host.Connect(ctx, *relayInfo); err != nil {
		LogError("Connect relay node fail: %v", relayInfo)
		return err
	}

	rCtx, _ := context.WithTimeout(p.Context, 15*time.Second)

	// 注册到中继节点，允许中继节点代理请求
	if _, err := client.Reserve(rCtx, p.Host, *relayInfo); err != nil {
		LogError("Reserve fail: %v", relayInfo)
		return err
	}

	for _, raddr := range relayInfo.Addrs {
		a := fmt.Sprintf("%s/p2p/%s/p2p-circuit/p2p/%s", raddr.String(), relayInfo.ID, p.Host.ID())
		LogInfo("Your address via relay is %s", url.QueryEscape(a))
	}

	return nil
}

func (p *Chat) AddContacts(s network.Stream) {
	ctx := context.WithoutCancel(p.Context)
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	newConcats := &Contacts{
		ID:         s.ID(),
		Name:       s.Conn().ID(),
		Context:    ctx,
		Stream:     s,
		ReadWriter: rw,
	}

	p.AddressBook = append(p.AddressBook, newConcats)
	p.AddressNotify <- newConcats
}

func (p *Chat) Send() {
	for sendData := range p.SendBox {
		LogInfo("sendData=%s\n\n", sendData)

		if sendData == "" || sendData == "\n" {
			continue
		}

		sendData = strings.Trim(sendData, " ")
		sendData = strings.Trim(sendData, "\n")

		if sendData[0] == '/' {
			cmd := strings.Split(sendData, " ")

			switch cmd[0] {
			case "/connect":
				if len(cmd) < 2 {
					LogError("Command.Parse.Err: raw=%s", sendData)
					break
				}

				address := strings.TrimRight(cmd[1], "\n")
				address = strings.TrimRight(address, " ")
				err := p.NewSession(address)

				if err != nil {
					LogError("Command.connect.Err: raw=%s", err.Error())
				}
			case "/relay":
				if len(cmd) < 2 {
					LogError("Command.Parse.Err: raw=%s", sendData)
					break
				}

				address := strings.TrimRight(cmd[1], "\n")
				address = strings.Trim(address, " ")
				err := p.NewRelay(address)

				if err != nil {
					LogError("Command.connect.Err: raw=%s", err.Error())

				}
			default:
				LogError("Unknow.Command %s", cmd[0])
			}
		} else {
			LogInfo("You: %s\n", sendData)
			for _, c := range p.AddressBook {
				c := c
				go func() {
					if c.Stream.Conn().IsClosed() {
						return
					}

					fmt.Printf("%+v\n\n", []byte(sendData+"\n"))
					n, err := c.ReadWriter.WriteString(fmt.Sprintf("%s\n", sendData))

					if err != nil {
						LogError("Write to err %s", err.Error())
					} else {
						LogInfo("Write %d bytes", n)
					}

					c.ReadWriter.Flush()
				}()
			}
		}
	}
}

func (p *Chat) Receive() {
	// msgBox := make(chan Message, 1000)

	go func() {
		for ab := range p.AddressNotify {
			LogInfo("Got new listener=%s", ab.Name)
			go func(u *Contacts) {
				for {
					if u.Stream.Conn().IsClosed() {
						LogInfo("Conn closed, listener=%s", ab.Name)
						return
					}

					content, err := u.ReadWriter.ReadString('\n')

					if err != nil {
						LogError("Listen.Err: speaker=%s, err=%s", u.Name, err.Error())
						return
					}

					content = strings.TrimRight(content, "\n")
					content = strings.Trim(content, " ")

					LogError("Got msg=%s, speaker=%s", content, u.Name)
				}
			}(ab)
		}
	}()
}
