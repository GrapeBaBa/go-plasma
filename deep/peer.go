// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package deep

import (
	"io"
	"net"

	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
)

// Serializable information about a Peer. Sufficient to build `etcdRaft.Peer`
// or `discover.Node`.
type Address struct {
	raftId   uint16
	nodeId   discover.NodeID
	ip       net.IP
	p2pPort  uint16
	raftPort uint16
}

func newAddress(raftId uint16, raftPort uint16, node *discover.Node) *Address {
	return &Address{
		raftId:   raftId,
		nodeId:   node.ID,
		ip:       node.IP,
		p2pPort:  node.TCP,
		raftPort: raftPort,
	}
}

// A peer that we're connected to via both raft's http transport, and ethereum p2p
type Peer struct {
	address *Address       // For raft transport
	p2pNode *discover.Node // For ethereum transport
}

func NewPeer(a *Address, p *discover.Node) *Peer {
	return &Peer{
		address: a,
		p2pNode: p,
	}
}

func (p *Peer) GetP2PNode() *discover.Node    { return p.p2pNode }
func (p *Peer) GetAddress() *Address          { return p.address }
func (a *Address) GetIP() net.IP              { return a.ip }
func (a *Address) GetRaftId() uint16          { return a.raftId }
func (a *Address) GetNodeId() discover.NodeID { return a.nodeId }
func (a *Address) GetP2PPort() uint16         { return a.p2pPort }
func (a *Address) GetRaftPort() uint16        { return a.raftPort }

func (addr *Address) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{addr.raftId, addr.nodeId, addr.ip, addr.p2pPort, addr.raftPort})
}

func (addr *Address) DecodeRLP(s *rlp.Stream) error {
	// These fields need to be public:
	var temp struct {
		RaftId   uint16
		NodeId   discover.NodeID
		Ip       net.IP
		P2pPort  uint16
		RaftPort uint16
	}

	if err := s.Decode(&temp); err != nil {
		return err
	} else {
		addr.raftId, addr.nodeId, addr.ip, addr.p2pPort, addr.raftPort = temp.RaftId, temp.NodeId, temp.Ip, temp.P2pPort, temp.RaftPort
		return nil
	}
}

// RLP Address encoding, for transport over raft and storage in LevelDB.

func (addr *Address) toBytes() []byte {
	size, r, err := rlp.EncodeToReader(addr)
	if err != nil {
		panic(fmt.Sprintf("error: failed to RLP-encode Address: %s", err.Error()))
	}
	var buffer = make([]byte, uint32(size))
	r.Read(buffer)

	return buffer
}

func bytesToAddress(bytes []byte) *Address {
	var addr Address
	if err := rlp.DecodeBytes(bytes, &addr); err != nil {
		log.Fatalf("failed to RLP-decode Address: %v", err)
	}
	return &addr
}
