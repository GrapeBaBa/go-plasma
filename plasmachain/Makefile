.PHONY:	plasmachunkstore statedb tokenmanager

test:
	@echo "test all."
	@echo "test plasmachunkstore."
	-go test -v plasmachunkstore_test.go
	@echo "test statedb."
	-go test -v ./statedb_test.go
	@echo "test storage."
	-go test -v ./storage_test.go

statedb:
	@echo "test statedb."
	-go test -v ./statedb_test.go

storage:
	@echo "test storage."
	-go test -v ./storage_test.go
