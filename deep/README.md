Steps for setting up a RAFT cluster -- plasma, sql, nosql


For raft minimum 4 nodes 
# sql
```
mkdir -p /root/sql/bin/
mkdir -p /root/sql/qdata/dd
cp sql /root/sql/bin/
touch /root/sql/qdata/dd/static-nodes.json
```
```
nohup /root/sql/bin/sql \
--datadir /root/sql/qdata/dd \
--nodiscover \
--rpc \
--rpcaddr 0.0.0.0 \
--rpcapi admin,db,eth,debug,miner,net,shh,txpool,personal,web3,quorum,sql,raft \
--emitcheckpoints \
--raftport 50404 \
--rpcport 22003 \
--port 21003 \
--rpccorsdomain=* \
--rpcvhosts=* \
--plasmaaddr 'cloudstore.wolk.com' \
--plasmaport 80 \
--unlock 0 \
--verbosity 4 \
--raft \
--blockchainid 77 \
2>> /root/sql/qdata/sql.log &
```
This will fail
take the nodeid from the log file of all the nodes and enter them to static-nodes.json
```
[  
    "enode://f413da51dc4b32ea419307a6c30da5b6d44790ea1cb91b340d458c745a19c7602771371120c5fd67e73ebce824133215e53eee09e6e42f80cf7ffb46eb55d459@47.91.105.205:21002?discport=0&raftport=50403",
    "enode://97c77d94f3ef389073d981d56a5a85a611509eada305d0be8a5f9fc3decf2042f3cc024b432947e1b6ff81fcdba9fedd3d63b5e7bdb7d5f16b2a5220ced93439@23.97.179.160:21002?discport=0&raftport=50403",
    "enode://e752b5a3d2b5a25d895b63074d99539c1715d3d5d3c424917a92ea0b8147df90488c9f40b9a41b01107628a721f290a3f97222ad031ea20e1ff312c5e796e9d6@18.231.4.13:21002?discport=0&raftport=50403",
    "enode://fbf627854c8c8fe924575d2fa2a6c24ce579f8d380c77431f3e5cee0cde0cb653a95cb473b4957ce3cf2cd6e466210b4f1b82b92d5866c680bfb54c9af3c391b@52.221.179.3:21002?discport=0&raftport=50403"
]
```
and start the again

# nosql
```
mkdir -p /root/nosql/bin/
mkdir -p /root/nosql/qdata/dd
cp nosql /root/nosql/bin/
touch /root/nosql/qdata/dd/static-nodes.json
```
```

nohup /root/nosql/bin/nosql \
--datadir /root/nosql/qdata/dd \
--nodiscover \
--rpc \
--rpcaddr 0.0.0.0 \
--rpcapi admin,db,eth,debug,miner,net,shh,txpool,personal,web3,quorum,nosql \
--emitcheckpoints \
--raftport 50403 \
--rpcport 22002 \
--port 21002 \
--plasmaaddr 'cloudstore.wolk.com' \
--plasmaport 80 \
--unlock 0 \
--verbosity 6 \
--raft \
2>> /root/nosql/qdata/nosql.log &
```
This will fail
take the nodeid from the log file of all the nodes and enter them to static-nodes.json
```
[  
    "enode://f413da51dc4b32ea419307a6c30da5b6d44790ea1cb91b340d458c745a19c7602771371120c5fd67e73ebce824133215e53eee09e6e42f80cf7ffb46eb55d459@47.91.105.205:21002?discport=0&raftport=50403",
    "enode://97c77d94f3ef389073d981d56a5a85a611509eada305d0be8a5f9fc3decf2042f3cc024b432947e1b6ff81fcdba9fedd3d63b5e7bdb7d5f16b2a5220ced93439@23.97.179.160:21002?discport=0&raftport=50403",
    "enode://e752b5a3d2b5a25d895b63074d99539c1715d3d5d3c424917a92ea0b8147df90488c9f40b9a41b01107628a721f290a3f97222ad031ea20e1ff312c5e796e9d6@18.231.4.13:21002?discport=0&raftport=50403",
    "enode://fbf627854c8c8fe924575d2fa2a6c24ce579f8d380c77431f3e5cee0cde0cb653a95cb473b4957ce3cf2cd6e466210b4f1b82b92d5866c680bfb54c9af3c391b@52.221.179.3:21002?discport=0&raftport=50403"
]
```
and start the again

# cloudstore

```
mkdir -p /root/cloudstore/bin
cp cloudstore /root/cloudstore/bin/
```

```
nohup /root/cloudstore/bin/cloudstore \
--rpc \
--rpcaddr 0.0.0.0 \
--rpcport 8545 \
--rpccorsdomain=* \
--rpcvhosts=* \
--rpcapi personal,db,eth,net,web3,swarmdb,plasma,admin,cloudstore \
--verbosity 6 \
2>> /root/cloudstore/cloudstore.log &
```
