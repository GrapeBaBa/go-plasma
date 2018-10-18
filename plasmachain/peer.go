// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package plasmachain

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/wolkdb/go-plasma/deep"
	set "gopkg.in/fatih/set.v0"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	maxKnownPlasmaTxs    = 32768 // Maximum app transactions hashes to keep in the known list (prevent DOS)
	maxKnownAnchorTxs    = 32768 // Maximum app transactions hashes to keep in the known list (prevent DOS)
	maxKnownPlasmaBlocks = 1024  // Maximum app block hashes to keep in the known list (prevent DOS)
	handshakeTimeout     = 5 * time.Second
)

// PeerInfo represents a short summary of the Ethereum sub-protocol metadata known
// about a connected peer.
type PeerInfo struct {
	Version int `json:"version"` // Ethereum protocol version negotiated
}

type peer struct {
	id string

	*p2p.Peer
	rw p2p.MsgReadWriter

	version  int         // Protocol version negotiated
	forkDrop *time.Timer // Timed connection dropper if forks aren't validated in time

	lock sync.RWMutex

	knownPlasmaBlocks *set.Set // Set of block hashes known to be known by this peer
	knownPlasmaTxs    *set.Set // Set of block hashes known to be known by this peer
	knownAnchorTxs    *set.Set // Set of block hashes known to be known by this peer
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	id := p.ID()

	return &peer{
		Peer:              p,
		rw:                rw,
		version:           version,
		id:                fmt.Sprintf("%x", id[:8]),
		knownPlasmaBlocks: set.New(),
		knownPlasmaTxs:    set.New(),
		knownAnchorTxs:    set.New(),
	}
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	return &PeerInfo{
		Version: p.version,
	}
}

// MarkPlasmaBlock marks a block as known for the peer, ensuring that the block will never be propagated to this particular peer.
func (p *peer) MarkPlasmaBlock(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known block hash
	for p.knownPlasmaBlocks.Size() >= maxKnownPlasmaBlocks {
		p.knownPlasmaBlocks.Pop()
	}
	p.knownPlasmaBlocks.Add(hash)
}

// MarkPlasmaTransaction marks a transaction as known for the peer, ensuring that it will never be propagated to this particular peer.
func (p *peer) MarkPlasmaTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownPlasmaTxs.Size() >= maxKnownPlasmaTxs {
		p.knownPlasmaTxs.Pop()
	}
	p.knownPlasmaTxs.Add(hash)
}

// MarkAnchorTransaction marks a transaction as known for the peer, ensuring that it will never be propagated to this particular peer.
func (p *peer) MarkAnchorTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownAnchorTxs.Size() >= maxKnownAnchorTxs {
		p.knownAnchorTxs.Pop()
	}
	p.knownAnchorTxs.Add(hash)
}

func (p *peer) SendPlasmaTransactions(txs PlasmaTransactions) error {
	for _, tx := range txs {
		p.knownPlasmaTxs.Add(tx.Hash())

		// ---LOG---
		time := time.Now()
		timeStr := fmt.Sprintf("%s", time) // 2018-06-21 14:56:39.621701678 -0700 PDT m=+54.123692278
		timeStr = string(timeStr[0:23])    // 2018-06-21 14:56:39.621

		log.Debug("[(p *peer) SendPlasmaTransactions] Send Plasma Transaction", "Time", timeStr, "Peer", p.ID(), "Hash", tx.Hash(), "tx", tx)
		// ---LOG---

	}
	return p2p.Send(p.rw, PlasmaTxMsg, txs)
}

func (p *peer) SendAnchorTransactions(txs deep.AnchorTransactions) error {
	for _, tx := range txs {
		p.knownAnchorTxs.Add(tx.Hash())
		log.Debug("Sending AnchorTX to Peer", "AnchorTx", tx.Hash(), "Peer", p.ID())
	}
	return p2p.Send(p.rw, AnchorTxMsg, txs)
}

// SendNewBlock propagates an entire block to a remote peer.
func (p *peer) SendNewPlasmaBlock(block *Block) error {

	// ---LOG---
	time := time.Now()
	timeStr := fmt.Sprintf("%s", time) // 2018-06-21 14:56:39.621701678 -0700 PDT m=+54.123692278
	timeStr = string(timeStr[0:23])    // 2018-06-21 14:56:39.621

	log.Debug("[(p *peer) SendNewPlasmaBlock] Send Plasma Block", "Time", timeStr, "Peer", p.ID(), "Block", block, "msg", NewPlasmaBlockMsg)
	// ---LOG---

	p.knownPlasmaBlocks.Add(block.Hash())

	sendErr := p2p.Send(p.rw, NewPlasmaBlockMsg, block)
	if sendErr != nil {
		log.Debug("Error Sending Message", "error", sendErr)
		return sendErr
	}
	return nil
}

// statusData is the network packet for the status message.
type statusData struct {
	ProtocolVersion uint32
	NetworkId       uint64
}

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *peer) Handshake(network uint64) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	var status statusData // safe to read after two values have been received from errc

	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, &statusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       network,
		})
	}()
	go func() {
		errc <- p.readStatus(network, &status)
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}

	return nil
}

func (p *peer) readStatus(network uint64, status *statusData) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.NetworkId != network {
		return errResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("eth/%2d", p.version),
	)
}

// peerSet represents the collection of active peers currently participating in
// the Ethereum sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

func (ps *peerSet) GetPeers() map[string]*peer {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	return ps.peers
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutPlasmaBlock(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownPlasmaBlocks.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutAnchorTx(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownAnchorTxs.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutPlasmaTx(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownPlasmaTxs.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}
