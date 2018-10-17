.PHONY: plasma cloudstore

GOBIN = $(shell pwd)/build/bin
GO ?= latest

plasma:
	build/env.sh go run build/ci.go install ./cmd/plasma
	@echo "Done building."
	@echo "Run '$(GOBIN)/plasma --rpc --rpcaddr \"localhost\" --rpcport 8505 --rpcapi \"admin, personal,db,eth,net,web3,swarmdb,plasma\"' to launch plasma."

plasmachaintest:
	@echo "test PlasmaChain."
	-go test -v ./plasmachain
