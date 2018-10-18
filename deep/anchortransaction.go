// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package deep

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

type AnchorTransactions []*AnchorTransaction

type AnchorTransaction struct {
	BlockChainID uint64       `json:"blockchainID"    gencodec:"required"`
	BlockNumber  uint64       `json:"blocknumber"     gencodec:"required"`
	BlockHash    *common.Hash `json:"blockhash"       gencodec:"required"`
	Extra        Ownership    `json:"extra"`
	Sig          []byte       `json:"sig"             gencodec:"required"`
}

type Ownership struct {
	AddedOwners   []common.Address `json:"addedOwners"   gencodec:"required"`
	RemovedOwners []common.Address `json:"removedOwners" gencodec:"required"`
}

//go:generate gencodec -type AnchorTransaction -field-override anchorTransactionMarshaling -out anchorTransaction_json.go

//marshalling store external type, if different
type anchorTransactionMarshaling struct {
	BlockChainID hexutil.Uint64
	BlockNumber  hexutil.Uint64
	Sig          hexutil.Bytes
}

//StorageRootforSMTOfChunkIDs -- Iterate over all the chunks referenced by the storage root

/*
func (o *Ownership) EncodeRLP(w io.Writer) (err error) {
	if len(o.AddedOwners) == 0 && len(o.RemovedOwners) == 0 {
		err = rlp.Encode(w, 0)

	} else {
		err = rlp.Encode(w, []interface{}{
			o.AddedOwners,
			o.RemovedOwners,
		})
	}
	return err
}

func (tx *AnchorTransaction) EncodeRLP(w io.Writer) (err error) {
	if tx == nil {
		err = rlp.Encode(w, []interface{}{
			tx.BlockChainID,
			tx.BlockNumber,
			tx.BlockHash,
			0,
			tx.Sig,
		})
	} else {

		data, err := rlp.EncodeToBytes(&tx.Extra)
		if err == nil {
			err = rlp.Encode(w, []interface{}{
				tx.BlockChainID,
				tx.BlockNumber,
				tx.BlockHash,
				data,
				tx.Sig,
			})
		}
	}
	return err
}

func (tx *AnchorTransaction) DecodeRLP(s *rlp.Stream) error {
	var externalAnchor struct {
		BlockChainID uint64
		BlockNumber  uint64
		BlockHash    *common.Hash
		Extra        []byte
		Sig          []byte
	}
	if err := s.Decode(&externalAnchor); err != nil {
		return err
	}
	var ownership Ownership
	if len(externalAnchor.Extra) > 0 {
		if err := rlp.DecodeBytes(externalAnchor.Extra, &ownership); err != nil {
			return err
		}
	}
	tx.BlockChainID = externalAnchor.BlockChainID
	tx.BlockNumber = externalAnchor.BlockNumber
	tx.BlockHash = externalAnchor.BlockHash
	tx.Extra = ownership
	tx.Sig = externalAnchor.Sig
	return nil
}

*/

// short hash
func (tx *AnchorTransaction) ShortHash() (hash common.Hash) {

	shortTX := &AnchorTransaction{
		BlockChainID: tx.BlockChainID,
		BlockNumber:  tx.BlockNumber,
		BlockHash:    tx.BlockHash,
		Extra:        tx.Extra,
		Sig:          make([]byte, 0),
	}
	enc, _ := rlp.EncodeToBytes(&shortTX)
	//fmt.Printf("Raw Msg: %x\n", enc)
	h := Keccak256(enc)
	return common.BytesToHash(h)
}

func NewAnchorTransaction(blockchainId uint64, blockNumber uint64, blockHash *common.Hash) *AnchorTransaction {
	return &AnchorTransaction{
		BlockChainID: blockchainId,
		BlockNumber:  blockNumber,
		BlockHash:    blockHash,
	}
}

func (tx *AnchorTransaction) AddOwner(addr common.Address) {
	o := tx.Extra
	if addressExist(o.RemovedOwners, addr) {
		o.RemovedOwners = remove(o.RemovedOwners, addr)
		o.AddedOwners = remove(o.AddedOwners, addr)
	} else {
		if !addressExist(o.AddedOwners, addr) {
			o.AddedOwners = append(o.AddedOwners, addr)
		}
	}
	tx.Extra = o
}

func (tx *AnchorTransaction) RemoveOwner(addr common.Address) {
	o := tx.Extra
	if addressExist(o.AddedOwners, addr) {
		o.RemovedOwners = remove(o.RemovedOwners, addr)
		o.AddedOwners = remove(o.AddedOwners, addr)
	} else {
		if !addressExist(o.RemovedOwners, addr) {
			o.RemovedOwners = append(o.RemovedOwners, addr)
		}
	}
	tx.Extra = o
}

func (tx *AnchorTransaction) ValidateExtraData() (err error) {
	addedOwners := removeDuplicates(tx.Extra.AddedOwners)
	removedOwners := removeDuplicates(tx.Extra.RemovedOwners)
	for _, addr := range addedOwners {
		if addressExist(removedOwners, addr) {
			return fmt.Errorf("Invalid Ownership modification")
		}
	}
	tx.Extra.AddedOwners = addedOwners
	tx.Extra.RemovedOwners = removedOwners
	return nil
}

func (tx *AnchorTransaction) ValidateAnchor() (err error) {
	err = tx.ValidateExtraData()
	if err != nil {
		return err
	}
	//only check for invalid sig length
	_, err = tx.GetSigner()
	if err != nil {
		return err
	}
	if tx.BlockChainID == 0 {
		return fmt.Errorf("BlockchainID 0 Not Allowed")
	}
	return nil
}

// WARNING: state/ownership is not checked by signTX
func (tx *AnchorTransaction) SignTx(priv *ecdsa.PrivateKey) (err error) {
	err = tx.ValidateExtraData()
	if err != nil {
		return err
	}

	//shortHash := tx.ShortHash()
	log.Info("AnchorTransaction:SignTX | Hash ", "shortHash", tx.ShortHash().Hex(), "signedHash", common.Bytes2Hex(signHash(tx.ShortHash().Bytes())), "tx", tx)
	sig, err := crypto.Sign(signHash(tx.ShortHash().Bytes()), priv)
	if err != nil {
		return err
	}
	tx.Sig = make([]byte, 65)
	copy(tx.Sig, sig)
	return nil
}

//recoverPlain
func (tx *AnchorTransaction) GetSigner() (common.Address, error) {
	recoveredPub, err := crypto.Ecrecover(signHash(tx.ShortHash().Bytes()), tx.Sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(recoveredPub) == 0 || recoveredPub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(recoveredPub[1:])[12:])
	return addr, nil
}

// full RLP-encoded byte sequence
/*
func BytesToAnchorTransaction(txbytes []byte) (ok bool, tx *AnchorTransaction) {
	var anchortx AnchorTransaction
	err := rlp.DecodeBytes(txbytes, &anchortx)
	if err != nil {
		return false, tx
	}
	return true, tx
}
*/

/*
func BytesToAnchorTransaction(txbytes []byte) (ok bool, tx *AnchorTransaction) {
	var anchortx AnchorTransaction
	err := rlp.Decode(bytes.NewReader(txbytes), &anchortx)
	if err != nil {
		return false, tx
	}
	return true, tx
}
*/

func (tx *AnchorTransaction) Bytes() (enc []byte) {
	enc, _ = rlp.EncodeToBytes(&tx)
	return enc
}

// full hash
func (tx *AnchorTransaction) Hash() common.Hash {
	h := Keccak256(tx.Bytes())
	return common.BytesToHash(h)
}

func (tx *AnchorTransaction) Hex() string {
	return fmt.Sprintf("%x", tx.Bytes())
}

func (tx *AnchorTransaction) Size() common.StorageSize {
	return 1
}

func (tx *AnchorTransaction) String() string {
	if tx != nil {
		//base64Sig := base64.StdEncoding.EncodeToString(tx.Sig)
		return fmt.Sprintf(`{"BlockChainID":"%x", "BlockNumber":"%x", "BlockHash":"%x", "Extra":"%v", "Sig":"%x"}`, tx.BlockChainID, tx.BlockNumber, tx.BlockHash, tx.Extra.Hex(), tx.Sig)
	} else {
		return fmt.Sprintf("{ }")
	}

}

func (extra *Ownership) Hex() string {
	return fmt.Sprintf("%x", extra.Bytes())
}

func (extra *Ownership) Bytes() (enc []byte) {
	enc, _ = rlp.EncodeToBytes(&extra)
	return enc
}

func addressExist(addrList []common.Address, addr common.Address) bool {
	for _, a := range addrList {
		if a == addr {
			return true
		}
	}
	return false
}

func remove(addrList []common.Address, addr common.Address) []common.Address {
	for i := len(addrList) - 1; i >= 0; i-- {
		if addrList[i] == addr {
			addrList = append(addrList[:i], addrList[i+1:]...)
		}
	}
	return addrList
}

func removeDuplicates(addrList []common.Address) []common.Address {
	existed := map[common.Address]bool{}
	for addr := range addrList {
		existed[addrList[addr]] = true
	}

	addrs := []common.Address{}
	for uniqueAddr, _ := range existed {
		addrs = append(addrs, uniqueAddr)
	}
	return addrs
}

func signHash(shortHash []byte) []byte {
	signedHash := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(shortHash), shortHash)
	return crypto.Keccak256([]byte(signedHash))
}
