// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package plasmachain

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/wolkdb/go-plasma/deep"
)

// Note:
// (1) sig based on: Keccak(RLP[TokenID,Denomination,PrevBlock,PrevOwner,Recipient,Allowance,Spent,nil]) (sig are included as nil)
// (2) what's included in SMT: Keccak(RLP[TokenID,Denomination,PrevBlock,PrevBlock,PrevOwner,Recipient,Allowance,Spent,Sig])
// Since PTransfer is deprecated, only one valid transaction per each tokenID can be included in SMT.
// Errors, however, do not get included at all.
type Transaction struct {
	TokenID      uint64          `json:"tokenID"      gencodec:"required"` // 8 byte uint64
	Denomination uint64          `json:"denomination" gencodec:"required"` // 8 byte uint64
	DepositIndex uint64          `json:"depositIndex" gencodec:"required"` // 8 byte uint64
	PrevBlock    uint64          `json:"prevBlock"    gencodec:"required"` // 32 byte uint256
	PrevOwner    *common.Address `json:"prevOwner"    gencodec:"required"` // 20 byte (Singer or depositor)
	Recipient    *common.Address `json:"recipient"    gencodec:"required"` // 20 byte (newOwner)
	Allowance    uint64          `json:"allowance"    gencodec:"required"` // 8 byte uint64, such that Denomination = V(token owner's balance) + A(operator's allowance) + S(operator's spent)
	Spent        uint64          `json:"spent"        gencodec:"required"` // 8 byte uint64
	Sig          []byte          `json:"sig"`                              // 65 byte RSV; the signing must be done on the first 3 pieces, but the merkleroot is based off hash of all 4 pieces
	balance      uint64          `json:"balance"      rlp:"-"`             // used locally (for now)
}

//# go:generate gencodec -type Transaction -field-override transactionMarshaling -out transaction_json.go

//marshalling store external type, if different
type transactionMarshaling struct {
	TokenID      hexutil.Uint64
	Denomination hexutil.Uint64
	DepositIndex hexutil.Uint64
	PrevBlock    hexutil.Uint64
	Allowance    hexutil.Uint64
	Spent        hexutil.Uint64
	Sig          hexutil.Bytes
}

type Transactions []interface{}
type PlasmaTransactions []*Transaction

func DecodeRLPTransaction(txbytes []byte) (tx *Transaction, err error) {
	var txo Transaction
	err = rlp.DecodeBytes(txbytes, &txo)
	if err != nil {
		return tx, err
	}
	return &txo, nil
}

func NewTransaction(t *Token, tinfo *TokenInfo, recipient *common.Address) *Transaction {
	return &Transaction{
		TokenID:      tokenKey(tinfo.DepositIndex, tinfo.Denomination, tinfo.Depositor),
		Denomination: tinfo.Denomination,
		DepositIndex: tinfo.DepositIndex,
		PrevBlock:    0, // t.LastBlock,
		PrevOwner:    &(t.Owner),
		Recipient:    recipient,
		Allowance:    t.Allowance,
		Spent:        t.Spent,
		balance:      t.Balance,
	}
}

func (tx *Transaction) String() string {
	if tx != nil {
		return fmt.Sprintf("{\"tokenID\":\"%x\", \"denomination\":\"%v\", \"depositIndex\":\"%v\",\"prevBlock\":\"%v\", \"prevOwner\":\"%x\", \"Recipient\":\"%x\", \"allowance\":\"%v\", \"spent\":\"%v\",\"balance\":\"%v\", \"sig\":\"%x\"}",
			tx.TokenID, tx.Denomination, tx.DepositIndex, tx.PrevBlock, tx.PrevOwner, tx.Recipient, tx.Allowance, tx.Spent, tx.balance, tx.Sig)
	} else {
		return fmt.Sprint("{}")
	}
}

func (tx *Transaction) Size() common.StorageSize {
	return 1
}

//recoverPlain
func (tx *Transaction) GetSigner() (common.Address, error) {
	//shortHash := tx.ShortHash()
	//signedHash := signHash(shortHash.Bytes())
	log.Info("GetSigner", "txbyte", tx.Hex(), "ShortHash", tx.ShortHash().Hex(), "signedHash", common.Bytes2Hex(signHash(tx.ShortHash().Bytes())))
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

// short hash
func (tx *Transaction) ShortHash() (hash common.Hash) {
	shortTX := &Transaction{
		TokenID:      tx.TokenID,
		Denomination: tx.Denomination,
		DepositIndex: tx.DepositIndex,
		PrevBlock:    tx.PrevBlock,
		PrevOwner:    tx.PrevOwner,
		Recipient:    tx.Recipient,
		Allowance:    tx.Allowance,
		Spent:        tx.Spent,
		// matches solidity -- important
		Sig: make([]byte, 0),
	}
	enc, _ := rlp.EncodeToBytes(&shortTX)
	h := deep.Keccak256(enc)

	return common.BytesToHash(h)
}

// full hash
func (tx Transaction) Hash() common.Hash {
	return rlpHash([]interface{}{
		tx.TokenID,
		tx.Denomination,
		tx.DepositIndex,
		tx.PrevBlock,
		tx.PrevOwner,
		tx.Recipient,
		tx.Allowance,
		tx.Spent,
		tx.Sig,
	})
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func (tx *Transaction) Hex() string {
	return fmt.Sprintf("%x", tx.Bytes())
}

func (tx *Transaction) Bytes() (enc []byte) {
	enc, _ = rlp.EncodeToBytes(&tx)
	return enc
}

// WARNING: state/ownership is not checked by signTX
func (tx *Transaction) SignTx(priv *ecdsa.PrivateKey) (err error) {
	// signature is based off a short hash (does not include Sig or Receipt)
	//shortHash := tx.ShortHash()
	sig, err := crypto.Sign(signHash(tx.ShortHash().Bytes()), priv)
	if err != nil {
		return err
	}
	tx.Sig = make([]byte, 65)
	copy(tx.Sig, sig)
	//signer, _ := tx.GetSigner()
	return nil
}

// Check if sig match prevowner or deposit's recipient
func (tx *Transaction) ValidateSig() (err error) {
	signer, err := tx.GetSigner()
	if err != nil {
		return err
	}

	if bytes.Compare(signer.Bytes(), tx.PrevOwner.Bytes()) != 0 {
		//return fmt.Errorf("Invalid Singer. Recovered:%s vs Expected:%s", signer.Hex(), tx.PrevOwner.Hex())  //external: should not expose expected
		return fmt.Errorf("Invalid Singer. Recovered:%s vs Expected:%s", signer.Hex(), tx.PrevOwner.Hex()) //internal case
	}

	if tx.PrevBlock == 0 {
		tokenID := tokenKey(tx.DepositIndex, tx.Denomination, *tx.Recipient)
		if tokenID != tx.TokenID {
			return fmt.Errorf("Invalid Deposit Recipient")
		}
	}
	return nil
}

// full RLP-encoded byte sequence
func BytesToTransaction(txbytes []byte) (ok bool, tx *Transaction) {
	tx, err := DecodeRLPTransaction(txbytes)
	if tx.Allowance+tx.Spent > tx.Denomination {
		tx.balance = 0
		return false, tx
	} else {
		tx.balance = tx.Denomination - tx.Allowance - tx.Spent
	}
	if err != nil {
		return false, tx
	}
	return true, tx
}

func getLastTransaction(history []*Transaction) (tx *Transaction) {
	if len(history) > 0 {
		return history[len(history)-1]
	} else {
		return nil
	}
}

// This gives context to the signed message and prevents signing of transactions.
func signHash(shortHash []byte) []byte {
	signedHash := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(shortHash), shortHash)
	return crypto.Keccak256([]byte(signedHash))
}
