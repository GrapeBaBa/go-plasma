// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.\
package plasmachain

import (
	"errors"
	"fmt"
	"sync"
	"time"
	//	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	//	"github.com/ethereum/go-ethereum/crypto"

	"github.com/wolkdb/go-plasma/deep"
)

// Constants to match up protocol versions and messages
const (
	plasma66 = 66
)

// Official short name of the protocol used during capability negotiation.
var ProtocolName = "plasma"

// Supported versions of the eth protocol (first is primary).
var ProtocolVersions = []uint{plasma66}

// Number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []uint64{5}

const ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message

// eth protocol message codes
const (
	// Plasma msg
	StatusMsg         = 0x01
	PlasmaTxMsg       = 0x02
	NewPlasmaBlockMsg = 0x03
	AnchorTxMsg       = 0x04
)

type errCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
)

// txChanSize is the size of channel listening to TxPreEvent.
// The number is referenced from the size of tx pool.
const (
	txChanSize = 4096
)

func (e errCode) String() string {
	return errorToString[int(e)]
}

// PlasmaTxPreEvent and AnchorTxPreEvent is posted when an transaction enters the  transaction pool.
type PlasmaTxPreEvent struct{ Tx *Transaction }
type AnchorTxPreEvent struct{ Tx *deep.AnchorTransaction }

var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code zzz",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrSuspendedPeer:           "Suspended peer",
}

// errIncompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
var errIncompatibleConfig = errors.New("incompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type ProtocolManager struct {
	appId       uint64
	networkId   uint64
	plasmaChain *PlasmaChain

	maxPeers int

	peers *peerSet

	SubProtocols []p2p.Protocol

	scope event.SubscriptionScope

	anchor_txCh  chan AnchorTxPreEvent
	anchor_txSub event.Subscription
	AnchorTxFeed event.Feed

	plasma_txCh  chan PlasmaTxPreEvent
	plasma_txSub event.Subscription
	eventMux     *event.TypeMux
	PlasmaTxFeed event.Feed

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	txsyncCh    chan *txsync
	quitSync    chan struct{}
	noMorePeers chan struct{}

	//channel for passing Block
	generatedPlasmaBlockSub *event.TypeMuxSubscription

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg     sync.WaitGroup
	txpool *PlasmaTxPool
}

// NewProtocolManager returns a new ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the ethereum network.
func NewProtocolManager(config *Config, networkId uint64, mux *event.TypeMux, plasmaChain *PlasmaChain) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		networkId:   networkId,
		eventMux:    mux,
		plasmaChain: plasmaChain,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer),
		txsyncCh:    make(chan *txsync),
		quitSync:    make(chan struct{}),
		//noMorePeers:  make(chan struct{}),
	}
	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Compatible; initialise the sub-protocol
		version := version // Closure for the run
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), p, rw)
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					return manager.handle(peer)
				case <-manager.quitSync:
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return manager.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := manager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errIncompatibleConfig
	}
	return manager, nil
}

func (pm *ProtocolManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing Ethereum peer", "peer", id)

	// Unregister the peer from the downloader and Ethereum peer set
	//pm.downloader.UnregisterPeer(id)
	if err := pm.peers.Unregister(id); err != nil {
		log.Error("Peer removal failed", "peer", id, "err", err)
	}
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

// SubscribePlasmaTxPreEvent registers a subscription of PlasmaTxPreEvent and
// starts sending event to the given channel.
func (pm *ProtocolManager) SubscribePlasmaTxPreEvent(ch chan<- PlasmaTxPreEvent) event.Subscription {
	return pm.scope.Track(pm.PlasmaTxFeed.Subscribe(ch))
}

// SubscribeAnchorTxPreEvent registers a subscription of AnchorTxPreEvent and
// starts sending event to the given channel.
func (pm *ProtocolManager) SubscribeAnchorTxPreEvent(ch chan<- AnchorTxPreEvent) event.Subscription {
	return pm.scope.Track(pm.AnchorTxFeed.Subscribe(ch))
}

func (pm *ProtocolManager) Start(maxPeers int) {
	log.Info("Starting Plasma protocol")
	pm.maxPeers = maxPeers
	pm.anchor_txCh = make(chan AnchorTxPreEvent, txChanSize)
	pm.anchor_txSub = pm.SubscribeAnchorTxPreEvent(pm.anchor_txCh)
	pm.plasma_txCh = make(chan PlasmaTxPreEvent, txChanSize)
	pm.plasma_txSub = pm.SubscribePlasmaTxPreEvent(pm.plasma_txCh)
	go pm.plasma_txBroadcastLoop()
	go pm.wolk_txBroadcastLoop()

	pm.generatedPlasmaBlockSub = pm.eventMux.Subscribe(NewGeneratedPlasmaBlockEvent{})
	go pm.generatedPlasmaBlockBroadcastLoop()
	// go pm.CoreLoop()

	// start sync handlers
	go pm.syncer()
	go pm.txsyncLoop()
}

func responsibleHost(hr int, minute int) string {
	hosts := []string{
		"yaron-phobos-a",
		"yaron-phobos-b",
		"yaron-phobos-c",
		"plasma-dubai-ali",
		"plasma-netherlands-azu",
		"plasma-saopaulo-aws",
		"plasma-singapore-aws",
		"plasma-victoria-azu",
		"plasma-shanghai-ali",
		"plasma-frankfurt-aws",
		"plasma-ireland-azu",
		"plasma-jakarta-ali",
		"plasma-mumbai-aws",
		"plasma-quebec-azu",
		"plasma-beijing-ali",
		"anand-sanmateo",
		"tapas-kolkata",
		"plasma-london-aws",
		"plasma-singapore-ali",
		"plasma-osaka-azu",
		"plasma-tokyo-aws",
		"plasma-hongkong-azu",
		"plasma-seoul-aws",
		"plasma-paris-aws",
		"plasma-chennai-azu",
		"plasma-kl-ali",
		"plasma-sydney-aws",
		"plasma-shenzhen-ali",
		"dev-ru",
	}
	host := hosts[(hr*60+minute)%(len(hosts))]
	return host

}

func (pm *ProtocolManager) CoreLoop() {
	//curOwner := 0
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case _ = <-ticker.C:
		}
	}
	log.Info("CoreLoop DONE!!!")
}

func (pm *ProtocolManager) Stop() {
	log.Info("Stopping Plasma protocol")

	pm.anchor_txSub.Unsubscribe()            // quits txBroadcastLoop
	pm.plasma_txSub.Unsubscribe()            // quits txBroadcastLoop
	pm.generatedPlasmaBlockSub.Unsubscribe() // quits blockBroadcastLoop

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	pm.peers.Close()

	// Wait for all peer handler goroutines and the loops to come down.
	pm.wg.Wait()

	log.Info("Ethereum protocol stopped")
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, rw)
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {

	// Ignore maxPeers if this is a trusted peer
	if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("Ethereum peer connected", "name", p.Name())

	// Execute the Ethereum handshake
	if err := p.Handshake(pm.networkId); err != nil {
		p.Log().Debug("Ethereum handshake failed", "err", err)
		return err
	}

	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("Ethereum peer registration failed", "err", err)
		return err
	}
	defer pm.removePeer(p.id)

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("Ethereum message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
	log.Debug("[(pm *ProtocolManager) handleMsg] Called TOP")
	//TODO: Fail out if Regional ID Missing peer.RegId
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	log.Debug("[(pm *ProtocolManager) handleMsg] Called", "code", msg.Code, "peer", p.ID())
	if err != nil {
		log.Debug("[(pm *ProtocolManager) handleMsg] Error", "peer", p.ID(), "Error:", err)
		return err
	}
	/*WOLK review ProtocolMaxMsgSize */
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	// Handle the message depending on its contents
	switch {
	case msg.Code == PlasmaTxMsg:

		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			// don't send this back to the peer that just sent it to you!
			p.MarkPlasmaTransaction(tx.Hash())

			// ---LOG---
			peerStr := fmt.Sprintf("%s", p.ID())
			peerStr = string(peerStr[0:10])
			time := time.Now()
			timeStr := fmt.Sprintf("%s", time) // 2018-06-21 14:56:39.621701678 -0700 PDT m=+54.123692278
			timeStr = string(timeStr[0:23])    // 2018-06-21 14:56:39.621

			log.Debug("[(pm *ProtocolManager) handleMsg] Received Plasma Transaction", "Time", timeStr, "Peer", p.ID(), "tx", tx)
			// ---LOG---
		}

		// put the received txs into the plasma_txCh channel
		for _, tx := range txs {
			pm.plasma_txCh <- PlasmaTxPreEvent{Tx: tx}
		}

	case msg.Code == AnchorTxMsg:
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*deep.AnchorTransaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			// don't send this back to the peer that just sent it to you!
			p.MarkAnchorTransaction(tx.Hash())
			log.Info("[(pm *ProtocolManager) handleMsg] MarkPlasmaTransaction", "tx", tx)
		}

		// put the received txs into the plasma_txCh channel
		for _, tx := range txs {
			pm.anchor_txCh <- AnchorTxPreEvent{Tx: tx}
		}

	case msg.Code == NewPlasmaBlockMsg:
		var plasmaBlock *Block
		if err := msg.Decode(&plasmaBlock); err != nil {
			log.Debug("[(pm *ProtocolManager) handleMsg] Error Decoding Block", "Error", err)
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		log.Debug(fmt.Sprintf("[(pm *ProtocolManager) handleMsg] Received Block is [%+v]", plasmaBlock))
		pm.plasmaChain.ProcessPlasmaBlock(plasmaBlock)

		// ---LOG---
		time := time.Now()
		timeStr := fmt.Sprintf("%s", time) // 2018-06-21 14:56:39.621701678 -0700 PDT m=+54.123692278
		timeStr = string(timeStr[0:23])    // 2018-06-21 14:56:39.621

		log.Debug("[(pm *ProtocolManager) handleMsg] Received Plasma Block", "Time", timeStr, "Peer", p.ID(), "Block", plasmaBlock, "msg", msg)
		// ---LOG---

		// put the received block into the eventMux
		//p.MarkPlasmaBlock(plasmaBlock.Hash())
		//pm.eventMux.Post(NewGeneratedPlasmaBlockEvent{Block: plasmaBlock})

		p.MarkPlasmaBlock(plasmaBlock.Hash())
		pm.BroadcastPlasmaBlock(plasmaBlock)
	default:
		log.Debug("[(pm *ProtocolManager) handleMsg] Code Not found", "Code:", msg.Code)
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

func (pm *ProtocolManager) GetPeers() []string {
	peers := pm.peers.GetPeers()
	//peersLen := pm.peers.Len()
	peersArr := make([]string, 0)
	for _, p := range peers {
		id := p.ID()
		enodeID := fmt.Sprintf("%x", id[:])
		remoteAddr := p.RemoteAddr() // net.Addr
		addr := fmt.Sprintf("%s", remoteAddr)
		enode := enodeID + "@" + addr
		peersArr = append(peersArr, enode)
	}
	log.Debug(fmt.Sprintf("WOLK: handler.go GetPeers() peersArr: %s ", peersArr))
	return peersArr
}

// BroadcastBlock will either propagate a block to a subset of it's peers, or
// will only announce it's availability (depending what's requested).
func (pm *ProtocolManager) BroadcastPlasmaBlock(block *Block) {
	time := time.Now()
	timeStr := fmt.Sprintf("%s", time)
	log.Debug("[(pm *ProtocolManager) BroadcastPlasmaBlock] BroadcastPlasmaBlock", "Time", timeStr)
	hash := block.Hash()
	peers := pm.peers.PeersWithoutPlasmaBlock(hash)

	// Send the plasma block to all of our peers
	for _, peer := range peers {
		log.Debug("[(pm *ProtocolManager) BroadcastPlasmaBlock] Sending to Peer", "peer", peer.ID())
		peer.SendNewPlasmaBlock(block)
	}
}

// BroadcastAnchorTx will propagate a transaction to all peers which are not known to already have the given transaction.
func (pm *ProtocolManager) BroadcastAnchorTx(hash common.Hash, tx *deep.AnchorTransaction) {
	log.Debug("Broadcasting Anchor Transaction")
	// Broadcast transaction to a batch of peers not knowing about it
	peers := pm.peers.PeersWithoutAnchorTx(hash)
	for _, peer := range peers {
		err := peer.SendAnchorTransactions(deep.AnchorTransactions{tx})
		if err != nil {
			log.Debug("[(pm *ProtocolManager) BroadcastAnchorTx] Error Encountered Sending Anchor Transaction", "Error", err)
		}
	}
	log.Trace("Broadcast anchor transaction", "hash", hash, "recipients", len(peers))
}

// BroadcastPlasmaTx will propagate a transaction to all peers which are not known to already have the given transaction.
func (pm *ProtocolManager) BroadcastPlasmaTx(hash common.Hash, tx *Transaction) {
	time := time.Now()
	timeStr := fmt.Sprintf("%s", time)
	log.Debug("Broadcasting Plasma Transaction", "Time", timeStr)
	// Broadcast transaction to a batch of peers not knowing about it
	peers := pm.peers.PeersWithoutPlasmaTx(hash)
	//FIXME include this again: peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for _, peer := range peers {
		err := peer.SendPlasmaTransactions(PlasmaTransactions{tx})
		if err != nil {
			log.Debug("[(pm *ProtocolManager) BroadcastPlasmaTx] Error Encountered Sending Plasma Transaction", "Error", err)
		}
	}
	log.Trace("Broadcast plasma transaction", "hash", hash, "recipients", len(peers))
}

func (self *ProtocolManager) generatedPlasmaBlockBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.generatedPlasmaBlockSub.Chan() {
		switch ev := obj.Data.(type) {
		case NewGeneratedPlasmaBlockEvent:
			self.BroadcastPlasmaBlock(ev.Block) // First propagate block to peers
		}
	}
}

func (self *ProtocolManager) wolk_txBroadcastLoop() {
	for {
		select {
		case event := <-self.anchor_txCh:
			log.Info("Encountered anchor_txCh event and will attempt to broadcast", "Event", event)
			self.BroadcastAnchorTx(event.Tx.Hash(), event.Tx)

			// Err() channel will be closed when unsubscribing.
		case <-self.anchor_txSub.Err():
			return
		}
	}
}

func (self *ProtocolManager) plasma_txBroadcastLoop() {
	for {
		select {
		case event := <-self.plasma_txCh:
			log.Info("Encountered plasma_txCh event and will attempt to broadcast")
			self.BroadcastPlasmaTx(event.Tx.Hash(), event.Tx)
		case <-self.plasma_txSub.Err():
			return
		}
	}
}

// NodeInfo represents a short summary of the Ethereum sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network uint64 `json:"network"` // Ethereum network ID
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (self *ProtocolManager) NodeInfo() *NodeInfo {
	return &NodeInfo{
		Network: self.networkId,
	}
}
