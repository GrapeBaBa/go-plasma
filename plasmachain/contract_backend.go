// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package plasmachain

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

func setConnection(endpointUrl string) (conn *ethclient.Client, err error) {
	conn, err = ethclient.Dial(endpointUrl)
	if err != nil {
		log.Debug("Failed to connect", "client", endpointUrl, "err", err)
		return conn, err
	} else {
		//fmt.Printf("Successfully connected to: %v\n", endpointUrl)
		return conn, err
	}
}

func setSession(contractAddr common.Address, endpointUrl string, key *ecdsa.PrivateKey) (session *RootChainSession, err error) {
	// Instantiate the contract {caller, transactor, filterer}, whitout
	// setting Pre-Configured CallOpts and TransactOpts
	conn, err := setConnection(endpointUrl)
	if err != nil {
		return session, err
	}
	contract, err := NewRootChain(contractAddr, conn)
	if err != nil {
		fmt.Printf("Failed to instantiate session to contract: %v", err)
		return session, err
	}

	authT := bind.NewKeyedTransactor(key)
	session = &RootChainSession{
		Contract: contract,
		CallOpts: bind.CallOpts{
			Pending: true,
		},
		TransactOpts: bind.TransactOpts{
			From:   authT.From,
			Signer: authT.Signer,
		},
	}
	return session, nil
}

func (self *PlasmaChain) DepositChannel() error {
	//optsF := &bind.WatchOpts{}
	sink := make(chan *RootChainDeposit)
	sub, err := self.session.Contract.WatchDeposit(nil, sink, nil, nil)
	if err != nil {
		fmt.Printf("Unable to open subscription: %s\n", err)
		return err
	}
loop:
	for {
		select {
		case deposit := <-sink:
			fmt.Printf("DepositChannel [%v] Deposit: Depositor %v | Index %v  TokenID %x (Denomination: %v)\n",
				time.Now().Format(time.RFC850), deposit.Depositor.Hex(), deposit.DepositIndex, deposit.TokenID, deposit.Denomination)
		case e := <-sub.Err():
			fmt.Printf("ERR: %s\n\n", e)
			break loop
		}
	}
	return nil
}

func (self *PlasmaChain) PublishedBlockChannel() error {
	//optsF := &bind.WatchOpts{}
	sink := make(chan *RootChainPublishedBlock)
	sub, err := self.session.Contract.WatchPublishedBlock(nil, sink, nil, nil)
	if err != nil {
		fmt.Printf("Unable to open subscription: %s\n", err)
		return err
	}
loop:
	for {
		select {
		case block := <-sink:
			fmt.Printf("PublishedBlockChannel [%v] [#%v] transactionRoot %x | CurrentDepositIndex %v\n",
				time.Now().Format(time.RFC850), block.Blknum, block.RootHash, block.CurrentDepositIndex)
		case e := <-sub.Err():
			fmt.Printf("ERR: %s\n\n", e)
			break loop
		}
	}
	return nil
}

func (self *PlasmaChain) publishBlock(transactionRoot common.Hash, blocknumber uint64, submitBlocks bool) (err error) {
	latestPublishedBlk, _ := self.session.CurrentBlkNum()
	latestPublishedRoot, err := self.session.ChildChain(latestPublishedBlk)
	if err != nil {
		fmt.Printf("unable to retrieve latest published root from rootContract: %v\n", err)
		return err
	}

	if latestPublishedBlk+1 != blocknumber {
		fmt.Printf("POTENTIAL ERR: Trying to publish non-sequential block: Last Published[#%v %x] | curr[#%v %s]\n\n", latestPublishedBlk, latestPublishedRoot, blocknumber, transactionRoot.Hex())
		//return err
	}

	if submitBlocks {
		tx, err := self.session.SubmitBlock(transactionRoot, blocknumber)
		//TODO: BAD! production code should not sleep!!!
		time.Sleep(5000 * time.Millisecond)
		if err != nil {
			fmt.Printf("block submission error: %v/n", err)
			return err
		} else {
			fmt.Println(tx)
		}
	}
	return nil
}

func (self *RootChainFilterer) getPublishedBlock() (pbs []*RootChainPublishedBlock, err error) {
	queriedBlock := make(map[uint64]bool)
	publishedBlocks, err := self.FilterPublishedBlock(nil, nil, nil)
	if err != nil {
		fmt.Printf("Failed to retrieve PublishedBlock: %v\n", err)
		return pbs, err
	}
	nextRoot := true
	for nextRoot {
		nextRoot = publishedBlocks.Next()
		if nextRoot {
			publishedBlock := publishedBlocks.Event
			if !queriedBlock[publishedBlock.Blknum] {
				queriedBlock[publishedBlock.Blknum] = true
				pbs = append(pbs, publishedBlock)
				fmt.Printf("PublishedBlock: #%v | Hash %x | DepIndex %v\n", publishedBlock.Blknum, publishedBlock.RootHash, publishedBlock.CurrentDepositIndex)
			}
		}
	}
	return pbs, nil
}

func (self *RootChainFilterer) getPastDeposit(dStart, dEnd uint64) (deposits []*RootChainDeposit, err error) {
	depositRange := makeIndexRange(dStart, dEnd)
	if len(depositRange) == 0 {
		return deposits, nil
	}
	fmt.Printf("Index range: %v\n", depositRange)
	batchDeposits, err := self.FilterDeposit(nil, depositRange, nil)
	if err != nil {
		return deposits, fmt.Errorf("Failutre to retrieve deposit")
	}

	nextDeposit := true
	for nextDeposit {
		nextDeposit = batchDeposits.Next()
		if nextDeposit {
			deposit := batchDeposits.Event
			deposits = append(deposits, deposit)
		}
	}
	return deposits, nil
}

func makeIndexRange(start, end uint64) []uint64 {
	a := []uint64{}
	for i := start; i <= end; i++ {
		a = append(a, i)
	}
	return a
}
