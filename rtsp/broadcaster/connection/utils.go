package connection

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"time"

	mrand "math/rand"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

func GetHostAddress(ha host.Host) string {
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", ha.ID()))
	addr := ha.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}
func createDHTForPeerDiscovery() []peer.AddrInfo {
	// Create a DHT for peer discovery
	var bootstrapPeersAddr []multiaddr.Multiaddr
	for _, s := range []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	} {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			panic(err)
		}
		bootstrapPeersAddr = append(bootstrapPeersAddr, ma)
	}
	bootstrapPeers := make([]peer.AddrInfo, len(bootstrapPeersAddr))
	for i, addr := range bootstrapPeersAddr {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(addr)
		bootstrapPeers[i] = *peerinfo
	}
	return bootstrapPeers

}
func getOptionEnableAutoRelayWithPeerSource(bootstrapPeers []peer.AddrInfo) libp2p.Option {
	peerSource := func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		ch := make(chan peer.AddrInfo, numPeers)
		go func() {
			defer close(ch)
			// Return bootstrap peers as potential relay candidates
			count := 0
			for _, p := range bootstrapPeers {
				if count >= numPeers {
					break
				}
				select {
				case ch <- p:
					count++
				case <-ctx.Done():
					return
				}
			}
		}()
		return ch
	}
	return libp2p.EnableAutoRelayWithPeerSource(peerSource)
}

func MakeEnhancedHost(ctx context.Context, listenPort int, insecure bool, randseed int64) (host.Host, *dht.IpfsDHT, error) {
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}
	prv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, nil, err
	}
	bootstrapPeers := createDHTForPeerDiscovery()
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),    // all interfaces
			fmt.Sprintf("/ip6/::/tcp/%d", listenPort),         // IPv6 support
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/ws", listenPort), //
			fmt.Sprintf("/ip6/::/tcp/%d/ws", listenPort),      //
		),
		libp2p.Transport(websocket.New),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Identity(prv),
		libp2p.EnableRelay(),        // Enable circuit relay
		libp2p.EnableHolePunching(), // Enable NAT hole punching
		libp2p.EnableNATService(),   // Enable NAT port mapping
		// libp2p.ForceReachabilityPrivate(), // Assume we're behind NAT, DHT conflict
		getOptionEnableAutoRelayWithPeerSource(bootstrapPeers),
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		libp2p.Security(tls.ID, tls.New),
	}

	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}
	host, err := libp2p.New(opts...)
	if err != nil {
		return nil, nil, err
	}
	kademliaDHT, err := dht.New(ctx, host, dht.BootstrapPeers(bootstrapPeers...))
	if err != nil {
		return nil, nil, err
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	return host, kademliaDHT, nil
}

type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi
}

func InitMDNS(peerhost host.Host, rendezvous string) chan peer.AddrInfo {
	// register with service so that we get notified about peer discovery
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)
	ser := mdns.NewMdnsService(peerhost, rendezvous, n)
	if err := ser.Start(); err != nil {
		panic(err)
	}
	return n.PeerChan
}

func MakeBasicHost(listenPort int, insecure bool, randseed int64) (host.Host, error) {
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}
	prv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
			fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", listenPort),
		),
		libp2p.Identity(prv),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
		libp2p.EnableNATService(),
	}
	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}
	return libp2p.New(opts...)
}

func AnnounceDHT(ctx context.Context, kademliaDHT *dht.IpfsDHT, rendezvous string) {
	// Create a proper CID from the rendezvous string
	hash := sha256.Sum256([]byte(rendezvous))
	mh, _ := multihash.EncodeName(hash[:], "sha2-256")
	rendezvousBytes := cid.NewCidV1(cid.Raw, mh)
	// Wait for DHT to be ready
	time.Sleep(5 * time.Second)
	fmt.Printf("Announcing on DHT with rendezvous: %s\n", rendezvous)

	// Announce ourselves as a provider for this rendezvous
	err := kademliaDHT.Provide(ctx, rendezvousBytes, true)
	if err != nil {
		fmt.Printf("Failed to announce on DHT: %v\n", err)
	} else {
		fmt.Println("Successfully announced on DHT!")
	}

	// Keep announcing periodically
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				err := kademliaDHT.Provide(ctx, rendezvousBytes, true)
				if err != nil {
					fmt.Printf("Failed to re-announce on DHT: %v\n", err)
				} else {
					fmt.Println("Re-announced on DHT")
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
