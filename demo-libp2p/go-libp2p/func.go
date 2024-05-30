package main

import (
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
)

func NewNode(ip string, port int, opts ...libp2p.Option) (libp2phost.Host, error) {
	addr := fmt.Sprintf("/ip4/%s/tcp/%d", ip, port)
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(addr)
	newOpts := make([]libp2p.Option, 0, len(opts)+1)
	newOpts = append(newOpts, libp2p.ListenAddrs(sourceMultiAddr))
	newOpts = append(newOpts, opts...)
	node, err := libp2p.New(newOpts...)

	return node, err
}

func NewRSAKey(r io.Reader, bits int) (crypto.PrivKey, crypto.PubKey, error) {

	privateKey, publicKey, err := crypto.GenerateKeyPairWithReader(crypto.Secp256k1, 2048, r)
	return privateKey, publicKey, err
}
