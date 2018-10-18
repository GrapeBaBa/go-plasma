.PHONY: plasma transaction deposit token tokeninfo account bloom block statedb storage storageencode anchorregister anchordemoupdate reateanchortransaction createremotestorage localsingle localbatch queuechunk getchunkcloudstore remotesingle remotebatch all test

GOBIN = $(shell pwd)/build/bin
GO ?= latest

plasma:
	build/env.sh go run build/ci.go install ./cmd/plasma
	@echo "Done building."
	@echo "\nTo launch plasma, Run:\n$(GOBIN)/plasma --rpc --rpcaddr 'localhost' --rpcport 8505 --rpcapi 'admin, personal,db,eth,net,web3,swarmdb,plasma' "

test: all
	@echo "Test All"
	-go test -v -run Transaction ./plasmachain
	-go test -v -run Deposit ./plasmachain
	-go test -v -run Token ./plasmachain
	-go test -v -run TokenInfo ./plasmachain
	-go test -v -run Account ./plasmachain
	-go test -v -run Bloom ./plasmachain
	-go test -v -run Block ./plasmachain
	-go test -v -run Statedb ./plasmachain
	-go test -v -run Storage ./plasmachain
	-go test -v -run StorageEncode ./plasmachain
	-go test -v -run PlasmaChainInternal ./plasmachain
	 -go test -v -run AnchorRegister ./deep
	 -go test -v -run AnchorDemoUpdate ./deep
	 -go test -v -run CreateAnchorTransaction ./deep
	 -go test -v -run CreateRemoteStorage ./deep
	 -go test -v -run LocalSingle ./deep
	 -go test -v -run LocalBatch ./deep
	 -go test -v -run QueueChunk ./deep
	 -go test -v -run GetChunkCloudstore ./deep
	 -go test -v -run RemoteSingle ./deep
	 -go test -v -run RemoteBatch ./deep

transaction:
	@echo "Test Transaction."
	-go test -v -run Transaction ./plasmachain

deposit:
	@echo "Test Deposit."
	-go test -v -run Deposit ./plasmachain

token:
	@echo "Test Token."
	-go test -v -run Token ./plasmachain

tokeninfo:
	@echo "Test TokenInfo."
	-go test -v -run TokenInfo ./plasmachain

account:
	@echo "Test Account."
	-go test -v -run Account ./plasmachain

bloom:
	@echo "Test Bloom."
	-go test -v -run Bloom ./plasmachain

block:
	@echo "Test Block."
	-go test -v -run Block ./plasmachain

statedb:
	@echo "Test Statedb."
	-go test -v -run Statedb ./plasmachain

storage:
	@echo "Test Storage."
	-go test -v -run Storage ./plasmachain

storageencode:
	@echo "Test StorageEncode."
	-go test -v -run StorageEncode ./plasmachain

plasmachaininternal:
	@echo "Test PlasmaChainInternal."
	-go test -v -run PlasmaChainInternal ./plasmachain

anchorregister:
	 @echo "Test AnchorRegister."
	 -go test -v -run AnchorRegister ./deep

anchordemoupdate:
	 @echo "Test AnchorDemoUpdate."
	 -go test -v -run AnchorDemoUpdate ./deep

createanchortransaction:
	 @echo "Test CreateAnchorTransaction."
	 -go test -v -run CreateAnchorTransaction ./deep

createremotestorage:
	 @echo "Test CreateRemoteStorage."
	 -go test -v -run CreateRemoteStorage ./deep

localsingle:
	 @echo "Test LocalSingle."
	 -go test -v -run LocalSingle ./deep

localbatch:
	 @echo "Test LocalBatch."
	 -go test -v -run LocalBatch ./deep

queuechunk:
	 @echo "Test QueueChunk."
	 -go test -v -run QueueChunk ./deep

getchunkcloudstore:
	 @echo "Test GetChunkCloudstore."
	 -go test -v -run GetChunkCloudstore ./deep

remotesingle:
	 @echo "Test RemoteSingle."
	 -go test -v -run RemoteSingle ./deep

remotebatch:
	 @echo "Test RemoteBatch."
	 -go test -v -run RemoteBatch ./deep
