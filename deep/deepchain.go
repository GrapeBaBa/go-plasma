package deep

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

type Block interface {
	Hash() common.Hash
	Root() common.Hash // TODO: study this
	Number() uint64
	Time() *big.Int
	ParentHash() common.Hash
	Transactions() []Transaction
	NumberU64() uint64
	Header() Header
	Body() Body
	Encode() ([]byte, error)
}

type Blocks []Block

type Transaction interface {
	Hash() common.Hash
}

type Transactions []Transaction

type Backend interface {
	BlockChain() BlockChain     // was: core.BlockChain
	TxPool() TxPool             // was *core.TxPool
	StorageLayer() StorageLayer // was *ethdb.Database
}

type PendingStateEvent struct{}
type NewMinedBlockEvent struct{ Block Block }
type TxPreEvent struct {
	Tx interface{}
}
type ChainHeadEvent struct {
	Block interface{}
}
type MintRequestEvent struct{}

type BlockChain interface {
	ApplyTransaction(StateDB, Transaction) error
	HasBlock(common.Hash, uint64) bool
	CurrentBlock() Block
	GetBlockByHash(common.Hash) (Block, error)
	GetBlockChainID() uint64
	GetBlockByNumber(rpc.BlockNumber) (Block, error)
	SubscribeChainHeadEvent(ch chan<- ChainHeadEvent) event.Subscription
	ValidateBody(Block) error
	GetBlock(common.Hash, uint64) (Block, error)
	StateAt(root common.Hash) (StateDB, error)
	InsertChain(blocks Blocks) (int, error)
	InsertChainForPOA(blocks Blocks) (int, error)
	SetHead(common.Hash) error
	SetHeadBlock(Block) error
	MakeBlock(common.Hash, Header, uint64, []Transaction) (Block, error) //NewBlock??
	Stop() error
	TxPool() TxPool
	WriteBlock(Block) error
	ChainType() string
	GetStorageLayer() StorageLayer
}

type StateDB interface {
	RevertToSnapshot(revid int) error
	Snapshot() int
	// Prepare(thash, bhash common.Hash, ti int)
	//TODO: Verify current assumption that headerhash is unique across different blockchains
	Commit(rs StorageLayer /* blockchainID, */, blockNumber uint64, parentHash common.Hash) (Header, error)
}

type TxPool interface {
	SubscribeTxPreEvent(chan<- TxPreEvent) event.Subscription
	SubscribeMintRequestEvent(chan<- MintRequestEvent) event.Subscription
	//Pending() (map[common.Address]Transactions, error)
	GetTransactions() Transactions
	RemoveTransaction(hash common.Hash) error
	Stop()
}

type Body interface {
}

type Header interface {
	Hash() common.Hash
	Number() uint64
	NumberU64() uint64
}

type ChainEvent struct {
	Block Block
	Hash  common.Hash
	//      Logs  []*types.Log
}

type StorageLayer interface {
	GetChunk(k []byte) (v []byte, ok bool, err error)
	SetChunk(k []byte, v []byte) (err error)
	//SetChunkBatch([]*sqlcommon.Chunk) (err error)
	QueueChunk(k []byte, v []byte) (err error)
	Flush() error
	FlushToLocal() error
	Has(k []byte) (bool, error)
	Delete(k []byte) error
	SendAnchorTransaction(*AnchorTransaction) error
	Close() error
	//Add CacheStore Stuff ?
}

type WolkConfig interface {
	GetPlasmaAddr() string
	GetPlasmaPort() uint64
	GetCloudstoreAddr() string
	GetCloudstorePort() uint64
	GetDataDir() string
	IsLocalMode() bool
}
