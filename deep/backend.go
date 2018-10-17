package deep

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type RaftPeer struct {
	*discover.Node
	RaftPort uint16
}
type RaftService struct {
	blockchain    BlockChain   // was core.Blockchain
	remoteStorage StorageLayer // was ethdb.Database
	txMu          sync.Mutex
	txPool        TxPool // was *core.TxPool

	raftProtocolManager *ProtocolManager
	//startPeers          []*discover.Node
	startPeers []*RaftPeer

	// we need an event mux to instantiate the blockchain
	eventMux *event.TypeMux
	minter   *minter
}

func New(ctx *node.ServiceContext, chainConfig *params.ChainConfig, raftId, raftPort uint16, joinExisting bool, blockTime time.Duration, mintTime time.Duration, startPeers []*RaftPeer, datadir string, remoteStorage StorageLayer, blockChain BlockChain, txPool TxPool, block Block) (*RaftService, error) {
	log.Debug(fmt.Sprintf("raft New port %d", raftPort))
	log.Debug(fmt.Sprintf("raft New port %v", startPeers))
	service := &RaftService{
		eventMux:      ctx.EventMux,
		remoteStorage: remoteStorage,
		blockchain:    blockChain,
		txPool:        txPool,
		startPeers:    startPeers,
	}

	service.minter = newMinter(chainConfig, service, blockTime, mintTime)

	var err error
	if service.raftProtocolManager, err = NewProtocolManager(raftId, raftPort, service.blockchain, service.eventMux, startPeers, joinExisting, datadir, service.minter, block); err != nil {
		return nil, err
	}
	return service, nil
}

// Backend interface methods:

func (service *RaftService) BlockChain() BlockChain { return service.blockchain }

func (service *RaftService) StorageLayer() StorageLayer { return service.remoteStorage }
func (service *RaftService) EventMux() *event.TypeMux   { return service.eventMux }
func (service *RaftService) TxPool() TxPool             { return service.txPool }

//func (service *RaftService) Downloader() Downloader         { return service.downloader }

// node.Service interface methods:

func (service *RaftService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

/*
func (service *RaftService) Protocols() []p2p.Protocol {
return []p2p.Protocol{
	Name: "deep",
	Version: 1,
	Length: 2,
	Run:

} }
*/
func (service *RaftService) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "raft",
			Version:   "1.0",
			Service:   NewPublicRaftAPI(service),
			Public:    true,
		},
	}
}

// Start implements node.Service, starting the background data propagation thread
// of the protocol.
func (service *RaftService) Start(p2pServer *p2p.Server) error {
	service.raftProtocolManager.Start(p2pServer)
	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the protocol.
func (service *RaftService) Stop() error {
	service.blockchain.Stop()
	service.raftProtocolManager.Stop()
	service.minter.stop()
	service.eventMux.Stop()

	//service.cloudstore.Close()

	log.Info("Raft stopped")
	return nil
}
