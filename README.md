
## go-plasma

Reference implementation of plasma in Go.  Wolk has augmented Plasma to support the *Deep BlockChains Architecture* described in this [paper](https://github.com/wolkdb/deepblockchains/blob/master/Deep_Blockchains.pdf).

Currently, Wolk is exploring a generalized "plasma cashflow" implementation, where network fees can flow between token owner and operator; in particular, we are extending the original {deposit, exit} schemes with partial withdraw functionality, which will enable operator to withdraw fees without forcing users to exist tokens on the mainNet.

## Background

To get started, we recommend reading:

* [Deep Blockchains: The Scalable Multilayer Blockchains](https://github.com/wolkdb/deepblockchains/blob/master/Deep_Blockchains.pdf)

* [Plasma Cash with Sparse Merkle Trees and Bloom filters](https://ethresear.ch/t/plasma-cash-with-sparse-merkle-trees-bloom-filters-and-probabilistic-transfers/2006)

# Plasma Node
Pre-build plasma binary can be found on our website [TBA]. Alternatively, you can build plasma binary from the source:
```
# Build plasma binary
$ make plasma
build/env.sh go run build/ci.go install ./cmd/plasma
>>> /usr/local/go/bin/go install -ldflags -X main.gitCommit=fdc5f7b81073c35f1fd36efe4f6ca1c0f2f4f0a6 -s -v ./cmd/plasma
...
Done building.
Run './build/bin/plasma --rpc --rpcaddr "localhost" --rpcport 8505 --rpcapi "admin, personal,db,eth,net,web3,swarmdb,plasma"' to launch plasma.
```

To get started:
```
# Run POA plasma node, starting RPC services at port 8505
$ ./build/bin/plasma --datadir /tmp/datadir3 --plasma.datadir /tmp/plasma3 --port 30303  --verbosity 4 --rpc --rpcaddr "localhost" --rpcport 8505 --rpcapi "personal,db,eth,net,web3,swarmdb,plasma"

# Start Ganache backend
sh ./plasmachain/contracts/RootChain/Util/init_ganache-cli.sh

# Start event-listner
node ./plasmachain/contracts/RootChain/listener/listener.js writeLog /plasmachain/contracts/RootChain/listener/events.log

# Deploy Contract
sh ./plasmachain/contracts/RootChain/Util/curl-json_init.sh

# Start Deposit
sh ./plasmachain/contracts/RootChain/Util/curl-json_deposit.sh

# If successful, the following deposits will be mined in child block #1

Deposit:  TokenID b437230feb2d24db | Denomination 1000000000000000000 | DepositIndex 0  (Depositor: 0xA45b77a98E2B840617e2eC6ddfBf71403bdCb683, TxHash: 0xd7e629ac78805d54faea00fd64e08af4d88c511827aa9ef7f5ead3945d7a527b)
Deposit:  TokenID 37b01bd3adfc4ef3 | Denomination 1000000000000000000 | DepositIndex 1  (Depositor: 0x82Da88C31E874C678D529ad51E43De3A4BAF3914, TxHash: 0x953285f46c56bf5c1f70a0d811b2ddf6ae00ea962f521f1ba20c7dd47d0c2b6f)
Deposit:  TokenID b76883d225414136 | Denomination 2000000000000000000 | DepositIndex 2  (Depositor: 0x3088666E05794d2498D9d98326c1b426c9950767, TxHash: 0xc2b98414a0261c7e4cdcf10e3b459501f78d69a626e09d6acf82189fdce57bac)
Deposit:  TokenID 09af84bc1208918b | Denomination 3000000000000000000 | DepositIndex 3  (Depositor: 0xBef06CC63C8f81128c26efeDD461A9124298092b, TxHash: 0x4fdcebb3247a9a715e416e68439e563e8faf57c804642441b93724d5b4fe0878)
Deposit:  TokenID 7c00dfa72e8832ed | Denomination 4000000000000000000 | DepositIndex 4  (Depositor: 0x74f978A3E049688777E6120D293F24348BDe5fA6, TxHash: 0xa268b3f36d3bbdb30b549f99fe0f7e7c35a2cd9598518a0c3116aa3ddc8fd68d)
...

```

*Note*: [go-Plasma core](https://github.com/wolkdb/go-plasma) are pending for security audit.

## RPC Interface
Plasma node serves its users via JSON-RPC 2.0. Most Plasma APIs can be tested either directly in the console or via RPC-over-HTTP. To get started, we recommend using the console, as it allows relaxed input types (where both Integers and HexString are accepted), whereas RPC-over-HTTP requires strict HexString due to the [gencodec]("https://github.com/fjl/gencodec") package we are currently using.

```
# Launch console
$ ./build/bin/plasma attach ~/Library/Ethereum/plasmachain.ipc
> plasma
{
    getAnchor: function(),
    getAnchorTransactionPool: function(),
    getPlasmaBalance: function(),
    getPlasmaBlock: function(),
    getPlasmaBloomFilter: function(),
    getPlasmaExitProof: function(),
    getPlasmaToken: function(),
    getPlasmaTransactionPool: function(),
    getPlasmaTransactionProof: function(),
    getPlasmaTransactionReceipt: function(),
    sendAnchorTransaction: function(),
    sendPlasmaTransaction: function(),
    sendRawAnchorTransaction: function(),
    sendRawPlasmaTransaction: function()
}
```
Currently, Public JSON-RPC API support the following functions:

* `getPlasmaBalance(address bytes20, blockNumber uint64|tag)` - get a list of tokenIDs, denominations, and total balance owned by address(account) and blockNumber. `blockNumber` accepts optional tag `"latest"` to automatically use the latest mind block when targeted blockNumber is unknown.

```
> plasma.getPlasmaBalance("0x3088666E05794d2498D9d98326c1b426c9950767", "latest");
{
    balance: 2000000000000000000,
    denomination: 2000000000000000000,
    tokens: ["0xb76883d225414136"]
}

curl -X POST --data  '{"jsonrpc":"2.0","id":"1","method":"plasma_getPlasmaBalance","params":["0x3088666E05794d2498D9d98326c1b426c9950767","latest"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "1",
    "result": {
        "account": {
            "tokens": ["0xb76883d225414136"],
            "denomination": "0x1bc16d674ec80000",
            "balance": "0x1bc16d674ec80000"
        }
    }
}
```


* `getPlasmaToken(tokenID uint64, blockNumber uint64|tag)` - get token by tokenID and targeted blockNumber. This can used by a node to verify token ownership when receiving a token from sender.

```
> plasma.getPlasmaToken("0x37b01bd3adfc4ef3" , 1)
{
    allowance: 0,
    balance: 1000000000000000000,
    denomination: 1000000000000000000,
    depositIndex: 1,
    owner: "0x82da88c31e874c678d529ad51e43de3a4baf3914",
    prevBlock: 1,
    spent: 0,
    tokenID: "0x37b01bd3adfc4ef3"
}

curl -X POST --data  '{"jsonrpc":"2.0","id":"2","method":"plasma_getPlasmaToken","params":["0x37b01bd3adfc4ef3","0x1"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "2",
    "result": {
        "token": {
            "denomination": "0xde0b6b3a7640000",
            "prevBlock": "0x1",
            "owner": "0x82da88c31e874c678d529ad51e43de3a4baf3914",
            "balance": "0xde0b6b3a7640000",
            "allowance": "0x0",
            "spent": "0x0"
        },
        "tokenInfo": {
            "depositIndex": "0x1",
            "denomination": "0xde0b6b3a7640000",
            "depositor": "0x82da88c31e874c678d529ad51e43de3a4baf3914",
            "tokenID": "0x37b01bd3adfc4ef3"
        }
    }
}

*Note: The API returns Token(dynamic), and TokenInfo(static) struct
```

* `getPlasmaBlock(blockNumber uint64|tag)` - get Plasma BlockInfo by blockNumber. Every Plasma block contains rootHash for each tries, which can be used to verify state transition and incoming token transfer. {`tokenRoot`, `accountRoot`, `l3ChainRoot`} are persisted roots that contain the latest full state, whereas {`transactionRoot`,`anchorRoot`} are built independently and sparse at every block. An empty SMT trie will have Root: `0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff` as level 64 DefaultHash.

```
> plasma.getPlasmaBlock("0x1")
{
  block: {
    accountRoot: "0xea57cc53a6c26fd29dbaa385138f50d9030471222cf679fa88a7274a84870bbb",
    anchorRoot: "0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff",
    blockNumber: "0x1",
    bloomID: "0x21fbaff17bd372e965d29f76026d63a6511187fd14c791acae9600079626edb0",
    headerHash: "0xb552a637d7be42099cce57e6066d982507abceceb3984346d14240cb2bbb0442",
    l3ChainRoot: "0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff",
    parentHash: "0x0aa4a98f607a18281ac968af64de7dec95615be72131c7c6c3152b59605d5da7",
    sig: "0xfe0459186a677fee2180bea65548fab50486a2700c697489706b935e5d62ea8d560a57741883f3db7e5d67814eb844f294bf9fdee025ea4776507dc0a7ba11a101",
    time: "0x5bbfd44f",
    tokenRoot: "0x3d3419dee60f8cc5a0130ed16b09561bdd41b6799245e26ffa43544269a4bebc",
    transactionRoot: "0x47671e5902ab9767b5915ea3402084280e689505e037897523e7bca6e90e733c"
  }
}

curl -X POST --data  '{"jsonrpc":"2.0","id":"3","method":"plasma_getPlasmaBlock","params":["0x1"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "3",
    "result": {
        "block": {
            "parentHash": "0x0aa4a98f607a18281ac968af64de7dec95615be72131c7c6c3152b59605d5da7",
            "blockNumber": "0x1",
            "time": "0x5bbfd44f",
            "bloomID": "0x21fbaff17bd372e965d29f76026d63a6511187fd14c791acae9600079626edb0",
            "transactionRoot": "0x47671e5902ab9767b5915ea3402084280e689505e037897523e7bca6e90e733c",
            "tokenRoot": "0x3d3419dee60f8cc5a0130ed16b09561bdd41b6799245e26ffa43544269a4bebc",
            "accountRoot": "0xea57cc53a6c26fd29dbaa385138f50d9030471222cf679fa88a7274a84870bbb",
            "l3ChainRoot": "0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff",
            "anchorRoot": "0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff",
            "sig": "0xfe0459186a677fee2180bea65548fab50486a2700c697489706b935e5d62ea8d560a57741883f3db7e5d67814eb844f294bf9fdee025ea4776507dc0a7ba11a101",
            "headerHash": "0xb552a637d7be42099cce57e6066d982507abceceb3984346d14240cb2bbb0442"
        }
    }
}

*Note: For speed optimization, Block Body is not queried nor returned by external API
```

* `getPlasmaBloomFilter(bloomID bytes32)` - get Bloom Filter by bloomID. Bloom Filter is required for a node to efficiently validate token transfers.

```
> plasma.getPlasmaBloomFilter("0x21fbaff17bd372e965d29f76026d63a6511187fd14c791acae9600079626edb0")
{
    filter: "00000000000000000000000000000010..."
}

curl -X POST --data  '{"jsonrpc":"2.0","id":"4","method":"plasma_getPlasmaBloomFilter","params":["0x21fbaff17bd372e965d29f76026d63a6511187fd14c791acae9600079626edb0"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "4",
    "result": {
        "filter": "00000000000000000000000000000010..."
    }
}
```

* `getPlasmaTransactionPool()`  - get a list of pending transactions from a node's plasma txpool, sorted by sender address, then by `tokenID` (and `prevBlock` number)

* `getPlasmaTransactionReceipt(txHash bytes32)`  - get transaction Info {receipt, transaction, included blockNumber} by txHash. The receipt can be used to check whether a transaction actually was successful; pending transaction will have receipt status of `0` until it's mined. The sender could send this back to the recipient, or the recipient could simply call the API to retrieve transaction status themselves.

```
> plasma.getPlasmaTransactionReceipt("0x953285f46c56bf5c1f70a0d811b2ddf6ae00ea962f521f1ba20c7dd47d0c2b6f");
{
    blockNumber: 1,
    plasmatransaction: {
        allowance: "0x0",
        denomination: "0xde0b6b3a7640000",
        depositIndex: "0x1",
        prevBlock: "0x0",
        prevOwner: "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683",
        recipient: "0x82da88c31e874c678d529ad51e43de3a4baf3914",
        sig: "0x62465828076576e5d2141a65b00883f96e3370086cd8d69db5076acd4cf941960458006d2be2875047a9699acb87e6707166fcd8be01fe29e3227008e30dc09100",
        spent: "0x0",
        tokenID: "0x37b01bd3adfc4ef3"
  },
    receipt: 1
}

curl -X POST --data  '{"jsonrpc":"2.0","id":"5","method":"plasma_getPlasmaTransactionReceipt","params":["0x953285f46c56bf5c1f70a0d811b2ddf6ae00ea962f521f1ba20c7dd47d0c2b6f"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "5",
    "result": {
        "plasmatransaction": {
            "tokenID": "0x37b01bd3adfc4ef3",
            "denomination": "0xde0b6b3a7640000",
            "depositIndex": "0x1",
            "prevBlock": "0x0",
            "prevOwner": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683",
            "recipient": "0x82da88c31e874c678d529ad51e43de3a4baf3914",
            "allowance": "0x0",
            "spent": "0x0",
            "sig": "0x62465828076576e5d2141a65b00883f96e3370086cd8d69db5076acd4cf941960458006d2be2875047a9699acb87e6707166fcd8be01fe29e3227008e30dc09100"
        },
        "blockNumber": 1,
        "receipt": 1
    }
}
```

* `getPlasmaTransactionProof(txHash bytes32)` - get inclusion proof { blockNumber, txbyte, proofByte } by txHash. An inclusion proof is required to start exit or challenge double spend on MainNet.

* `getPlasmaExitProof(tokenID uint64)` - get smart exitProof by tokenID. An smart exitProof includes `{prevTxBytes, prevProofByte, preBlk, TxBytes, ProofByte, Blk}`, which is required to start exit on MainNet.

```
> plasma.getPlasmaTransactionProof("0x953285f46c56bf5c1f70a0d811b2ddf6ae00ea962f521f1ba20c7dd47d0c2b6f");
{
    blockNumber: "0x1",
    proofByte: "0xe000000000000000be40d13045b13413417b2c2b6e0f23d336ba7ae5c9d521187ccc179f262507986c6a8bd539c1e788e8a9fbe3c42243426bad642bcbc14100923b4610fed832b0d77bd62537b7ed25202ba195360d56e3c8021109df6646f0f1cab6a6e130801a",
    tokenID: "0x37b01bd3adfc4ef3",
    txbyte: "0xf8838837b01bd3adfc4ef3880de0b6b3a7640000018094a45b77a98e2b840617e2ec6ddfbf71403bdcb6839482da88c31e874c678d529ad51e43de3a4baf39148080b84162465828076576e5d2141a65b00883f96e3370086cd8d69db5076acd4cf941960458006d2be2875047a9699acb87e6707166fcd8be01fe29e3227008e30dc09100"
}

curl -X POST --data  '{"jsonrpc":"2.0","id":"6","method":"plasma_getPlasmaTransactionProof","params":["0x953285f46c56bf5c1f70a0d811b2ddf6ae00ea962f521f1ba20c7dd47d0c2b6f"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "6",
    "result": {
        "blockNumber": "0x1",
        "proofByte": "0xe000000000000000be40d13045b13413417b2c2b6e0f23d336ba7ae5c9d521187ccc179f262507986c6a8bd539c1e788e8a9fbe3c42243426bad642bcbc14100923b4610fed832b0d77bd62537b7ed25202ba195360d56e3c8021109df6646f0f1cab6a6e130801a",
        "tokenID": "0x37b01bd3adfc4ef3",
        "txbyte": "0xf8838837b01bd3adfc4ef3880de0b6b3a7640000018094a45b77a98e2b840617e2ec6ddfbf71403bdcb6839482da88c31e874c678d529ad51e43de3a4baf39148080b84162465828076576e5d2141a65b00883f96e3370086cd8d69db5076acd4cf941960458006d2be2875047a9699acb87e6707166fcd8be01fe29e3227008e30dc09100"
    }
}
```

## Plasma Transaction
When one node sends a Plasma token to another:

1. To make plasma cash "usable", we have implemented `tokenStorage trie` that keeps tracks of every token's state transition in a persisted SMT tree. This enables sender to submit `sendPlasmaTransaction` without the need of providing the entire token history as inputs.  A plasma node will relay `sendPlasmaTransaction` request to its peers via `BroadcastPlasmaTransaction`, one of whom will be the leader (authority). leader (authority) verifies a transaction via the `tokenRoot` and will run `MintPlasmaBlock` within few seconds (e.g. after 5 seconds, or after 1000 transactions, whichever comes first). While pending transaction sits in the transaction pool, txHash is returned back to the sender immediately. Calling `getPlasmaTransactionReceipt` will return status `0` until the tx is mined.

2. New Blocks are created _after_ leader (authority) uses the `PlasmaChunkstore` abstraction to verifiably store chunks among different cloudstores. leader (authority) will then transmit `BroadcastBlock` event to all the plasma nodes and update `PlasmaTransactionReceip` status to `1`.
If all are successful, leader (authority) will publish `transactionRoot` and `blockNumber` to MainNet via `submitBlock`.

3. The sender, having seen its transaction "cleared" with the plasma operator (and seeing that it has a "proof of inclusion" on MainNet), can then send a "proof" to its fellow node by calling `getPlasmaTransactionProof` and sending the data with its own _SWAP_ protocol.  The recipient will then:
 * calling `getPlasmaTransactionReceipt` itself, retrieving the transaction
 * check the signature matches the sender
 * check the transaction has been included in the block by calling MainNet for the proof (this is free!)
 * download all Bloom filters, all the way back to the previous checkpoint
 * For false-positive case in Bloom filters or non-empty blocks, check for token history.
 * update its own Wallet


IMPORTANT: While transaction can be initiated without full token history, token owners should still keep the full history _independent from plasma operator_ to protect themselves against malicious operator.

* `sendPlasmaTransaction(plasmaTX)` - requires PlasmaTransaction type as input. This function can be called by token owners to Initiate token transfer:

* `sendRawPlasmaTransaction(txBytes)` - requires `rlpencode(plasmaTX)` as input.


```
# PlasmaTransaction Type
{
    "tokenID": uint64,
    "denomination": uint64,
    "depositIndex": uint64,
    "prevBlock": uint64,
    "prevOwner": address(bytes20),
    "recipient": address(bytes20),
    "allowance": uint64,
    "spent": uint64,
    "balance": uint64,
    "sig": bytes ([r,s,v], length 65)
}

Note: balance field is optional
```

```
# Initiate a Plasma Transaction on Layer 2
> plasma.sendPlasmaTransaction({"tokenID":0x37b01bd3adfc4ef3, "denomination": 1000000000000000000, "depositIndex": 1, "prevBlock": 1, "prevOwner": "0x82Da88C31E874C678D529ad51E43De3A4BAF3914", "recipient": "0x3088666E05794d2498D9d98326c1b426c9950767", "allowance": 100000000000000001, "spent": 200000000000000002, "sig": "0xffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600"});
"0x8dde5209f1c65c0dbfd88a75ea31160823184b410984765f0a4624e649215df1"

# Retrieve Pending PlasmaTxs from PlasmaTransactionPool
> plasma.getPlasmaTransactionPool()
{
  0x82da88c31e874c678d529ad51e43de3a4baf3914: [{
      allowance: "0x16345785d8a0001",
      denomination: "0xde0b6b3a7640000",
      depositIndex: "0x1",
      prevBlock: "0x1",
      prevOwner: "0x82da88c31e874c678d529ad51e43de3a4baf3914",
      recipient: "0x3088666e05794d2498d9d98326c1b426c9950767",
      sig: "0xffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600",
      spent: "0x2c68af0bb140002",
      tokenID: "0x37b01bd3adfc4ef3"
  }]
}

# Query PlasmaTransactionReceipt (After Block is Mined)
> plasma.getPlasmaTransactionReceipt("0x8dde5209f1c65c0dbfd88a75ea31160823184b410984765f0a4624e649215df1")
{
    blockNumber: 2,
    plasmatransaction: {
        allowance: "0x16345785d8a0001",
        denomination: "0xde0b6b3a7640000",
        depositIndex: "0x1",
        prevBlock: "0x1",
        prevOwner: "0x82da88c31e874c678d529ad51e43de3a4baf3914",
        recipient: "0x3088666e05794d2498d9d98326c1b426c9950767",
        sig: "0xffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600",
        spent: "0x2c68af0bb140002",
        tokenID: "0x37b01bd3adfc4ef3"
    },
    receipt: 1
}

# Retrieve Token State from Latest Block
> plasma.getPlasmaToken("0x37b01bd3adfc4ef3", "latest")
{
    allowance: 100000000000000001,
    balance: 699999999999999997,
    denomination: 1000000000000000000,
    depositIndex: 1,
    owner: "0x3088666e05794d2498d9d98326c1b426c9950767",
    prevBlock: 2,
    spent: 200000000000000002,
    tokenID: "0x37b01bd3adfc4ef3"
}

# Retrieve Latest Block
> plasma.getPlasmaBlock("latest");
{
    block: {
        accountRoot: "0x5c237e0b259471155074e667da1674037fd1b0ef0220063e524940b07d3f503f",
        anchorRoot: "0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff",
        blockNumber: "0x2",
        bloomID: "0xda0f9b66e614309fead8740a3f318fd4864a09462753b0db8ae5004c7d7bfa68",
        headerHash: "0xc87423332c165446eeeaf1a09eb041804ecf917a92e8d822272f654f5561d64d",
        l3ChainRoot: "0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff",
        parentHash: "0x9f942f75f71054c9e5cf56614755715bde7860cbacaad3ff8fc53966e238d4ee",
        sig: "0x61dcfca6436ac60d81c01d40602d9fd955205b8a98362a12c2abdf307f953d4c423ca36c180b4da68d1272c30c47d819dac9bc025ce98849a393abc66f63390b00",
        time: "0x5bbfd46d",
        tokenRoot: "0x16bc6ad0aa4cec83fdc3c7e3ae217afb9bc75fcb358f329debd54b8a00fa42f2",
        transactionRoot: "0xfa0e5f45f4f02e3ad034a495c408c449acbbffb4d38a21767b9a9af738a16c6a"
    }
}

# Retrieve Transaction Proof
> plasma.getPlasmaTransactionProof("0x8dde5209f1c65c0dbfd88a75ea31160823184b410984765f0a4624e649215df1");
{
    blockNumber: "0x2",
    proofByte: "0x0000000000000000",
    tokenID: "0x37b01bd3adfc4ef3",
    txbyte: "0xf8938837b01bd3adfc4ef3880de0b6b3a764000001019482da88c31e874c678d529ad51e43de3a4baf3914943088666e05794d2498d9d98326c1b426c995076788016345785d8a00018802c68af0bb140002b841ffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600"
}

# Retrieve PlasmaBalance by Owner's Address
plasma.getPlasmaBalance("0x3088666E05794d2498D9d98326c1b426c9950767", "latest");
{
    balance: 2699999999999999997,
    denomination: 3000000000000000000,
    tokens: ["0xb76883d225414136", "0x37b01bd3adfc4ef3"]
}
```

```
# Initiate a Plasma Transaction on Layer 2
curl -X POST --data '{"jsonrpc":"2.0","id":"7","method":"plasma_sendPlasmaTransaction","params":[{"tokenID":"0x37b01bd3adfc4ef3", "denomination": "0xde0b6b3a7640000", "depositIndex": "0x1", "prevBlock": "0x1", "prevOwner": "0x82Da88C31E874C678D529ad51E43De3A4BAF3914", "recipient": "0x3088666E05794d2498D9d98326c1b426c9950767", "allowance": "0x16345785d8a0001", "spent": "0x2c68af0bb140002", "sig": "0xffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600"}]}' -H 'content-type:application/json;' 'localhost:8505'
{
    "jsonrpc": "2.0",
    "id": "7",
    "result": "0x8dde5209f1c65c0dbfd88a75ea31160823184b410984765f0a4624e649215df1"
}

# Retrieve Pending PlasmaTxs from PlasmaTransactionPool
curl -X POST --data  '{"jsonrpc":"2.0","id":"8","method":"plasma_getPlasmaTransactionPool"}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "8",
    "result": {
        "0x82da88c31e874c678d529ad51e43de3a4baf3914": [
            {
                "tokenID": "0x37b01bd3adfc4ef3",
                "denomination": "0xde0b6b3a7640000",
                "depositIndex": "0x1",
                "prevBlock": "0x1",
                "prevOwner": "0x82da88c31e874c678d529ad51e43de3a4baf3914",
                "recipient": "0x3088666e05794d2498d9d98326c1b426c9950767",
                "allowance": "0x16345785d8a0001",
                "spent": "0x2c68af0bb140002",
                "sig": "0xffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600"
            }
        ]
    }
}

# Query PlasmaTransactionReceipt (After Block is Mined)
curl -X POST --data  '{"jsonrpc":"2.0","id":"9","method":"plasma_getPlasmaTransactionReceipt","params":["0x8dde5209f1c65c0dbfd88a75ea31160823184b410984765f0a4624e649215df1"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "9",
    "result": {
        "receipt": 1,
        "plasmatransaction": {
            "tokenID": "0x37b01bd3adfc4ef3",
            "denomination": "0xde0b6b3a7640000",
            "depositIndex": "0x1",
            "prevBlock": "0x1",
            "prevOwner": "0x82da88c31e874c678d529ad51e43de3a4baf3914",
            "recipient": "0x3088666e05794d2498d9d98326c1b426c9950767",
            "allowance": "0x16345785d8a0001",
            "spent": "0x2c68af0bb140002",
            "sig": "0xffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600"
        },
        "blockNumber": 2
    }
}

# Retrieve Token State from Latest Block
curl -X POST --data  '{"jsonrpc":"2.0","id":"10","method":"plasma_getPlasmaToken","params":["0x37b01bd3adfc4ef3","latest"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "10",
    "result": {
        "token": {
            "denomination": "0xde0b6b3a7640000",
            "prevBlock": "0x2",
            "owner": "0x3088666e05794d2498d9d98326c1b426c9950767",
            "balance": "0x9b6e64a8ec5fffd",
            "allowance": "0x16345785d8a0001",
            "spent": "0x2c68af0bb140002"
        },
        "tokenInfo": {
            "depositIndex": "0x1",
            "denomination": "0xde0b6b3a7640000",
            "depositor": "0x82da88c31e874c678d529ad51e43de3a4baf3914",
            "tokenID": "0x37b01bd3adfc4ef3"
        }
    }
}

# Retrieve Latest Block
curl -X POST --data  '{"jsonrpc":"2.0","id":"11","method":"plasma_getPlasmaBlock","params":["latest"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "11",
    "result": {
        "block": {
            "parentHash": "0x9f942f75f71054c9e5cf56614755715bde7860cbacaad3ff8fc53966e238d4ee",
            "blockNumber": "0x2",
            "time": "0x5bbfd46d",
            "bloomID": "0xda0f9b66e614309fead8740a3f318fd4864a09462753b0db8ae5004c7d7bfa68",
            "transactionRoot": "0xfa0e5f45f4f02e3ad034a495c408c449acbbffb4d38a21767b9a9af738a16c6a",
            "tokenRoot": "0x16bc6ad0aa4cec83fdc3c7e3ae217afb9bc75fcb358f329debd54b8a00fa42f2",
            "accountRoot": "0x5c237e0b259471155074e667da1674037fd1b0ef0220063e524940b07d3f503f",
            "l3ChainRoot": "0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff",
            "anchorRoot": "0xb992a50058a2812b0fc4fe1bbbfb3d8ffd476fb89391408212e00a7019e10eff",
            "sig": "0x61dcfca6436ac60d81c01d40602d9fd955205b8a98362a12c2abdf307f953d4c423ca36c180b4da68d1272c30c47d819dac9bc025ce98849a393abc66f63390b00",
            "headerHash": "0xc87423332c165446eeeaf1a09eb041804ecf917a92e8d822272f654f5561d64d"
        }
    }
}

# Retrieve Transaction Proof
curl -X POST --data  '{"jsonrpc":"2.0","id":"12","method":"plasma_getPlasmaTransactionProof","params":["0x8dde5209f1c65c0dbfd88a75ea31160823184b410984765f0a4624e649215df1"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "12",
    "result": {
        "blockNumber": "0x2",
        "proofByte": "0x0000000000000000",
        "tokenID": "0x37b01bd3adfc4ef3",
        "txbyte": "0xf8938837b01bd3adfc4ef3880de0b6b3a764000001019482da88c31e874c678d529ad51e43de3a4baf3914943088666e05794d2498d9d98326c1b426c995076788016345785d8a00018802c68af0bb140002b841ffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600"
    }
}

# Retrieve PlasmaBalance by Owner's Address
curl -X POST --data  '{"jsonrpc":"2.0","id":"13","method":"plasma_getPlasmaBalance","params":["0x3088666E05794d2498D9d98326c1b426c9950767","latest"]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "13",
    "result": {
        "account": {
            "tokens": [
                "0xb76883d225414136",
                "0x37b01bd3adfc4ef3"
            ],
            "denomination": "0x29a2241af62c0000",
            "balance": "0x257853b1dd8dfffd"
        }
    }
}
```




## Anchor Transaction

* BlockchainID Assignment - A valid token is required to create Layer 3 blockchain. L3 blockchainID is automatically assigned using the last 8 bytes of  `Keccak256(tokenID, blockchainNonce)`.  For example, the first L3 chain created by `0x37b01bd3adfc4ef3` will have blockchainID `0x69eb463bc4f6b2df`, the last 8 bytes of `Keccak256(37b01bd3adfc4ef3,0)`

* AnchorBlock - To accommodate L3's ultra high block frequency, a "pseudo" AnchorBlock is built to compress multiple chain-specific anchorTxs into one root hash. AnchorBlock itself is a *compacted*, *ordered* Merkle Tree and its root hash is stored at L3Chain SMT's leaf node.

* `getAnchorTransactionPool()`  - get a list of pending Anchor Transactions received by a plasma node (the Layer 2 operator). Anchor Transactions are sorted by L3 chain's blockchainID then by each L3 chain's blocknumber

* `sendAnchorTransaction(AnchorTX)` - requires AnchorTransaction type as input. This function is called by Layer3 Chain operators to submit their blockhashes to plasma nodes. AnchorTransactions can also be used to modify Layer3 chain permission by passing in non-empty extra.

* `sendRawAnchorTransaction(txBytes)` - requires `rlpencode(AnchorTX)` as input

```
# AnchorTransaction Type
{
    "blockchainID": uint64,
    "blocknumber": uint64,
    "blockhash": bytes32,
    "extra":{"addedOwners": ["addr1", "addr2",...], "removedOwners": Array[Addr]},
    "sig": bytes ([r,s,v], length 65)
}

WARNING: empty extra must be encoded as 'RLP([[],[]])' or 'c2c0c0' when there's no change for ownership
```


```
# Creates Layer3 Blockchain (AnchorTx with blocknumber #0)
> plasma.sendAnchorTransaction({"blockchainID":"0x69eb463bc4f6b2df","blocknumber":"0x0","blockhash":"0x03c85f1da84d9c6313e0c34bcb5ace945a9b12105988895252b88ce5b769f82b","sig":"0x68affbe7c2ad15fdec05427fc7fa94cb885e28525463ab2b04207b5972c6bb5e57c9596f98b5e1e5c8896e46b643a8734cfd30934272012c90081209dc15a0f300"})
"0x6808da8621d7c01021dbd98c37344e48418dd1967d6302d73a7b1c9341ca3be1"


# Add/Remove Owners for a Layer3 Blockchain (AnchorTx with non-empty extradata)
> plasma.sendAnchorTransaction({"blockchainID":"0x69eb463bc4f6b2df","blocknumber":"0x1","blockhash":"0x6d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4","extra":{"addedOwners":["0x3088666E05794d2498D9d98326c1b426c9950767"],"removedOwners":[]},"sig":"0x5a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201"})
"0xb6f91e2a3059e7a2d347d76d2e548f0a3c58068f0185f5dadcceb9580591657a"

# Retrieve Pending AnchorTxs from AnchorTransactionPool
> plasma.getAnchorTransactionPool()
{
  7632271216030954207: [{
      blockchainID: "0x69eb463bc4f6b2df",
      blockhash: "0x03c85f1da84d9c6313e0c34bcb5ace945a9b12105988895252b88ce5b769f82b",
      blocknumber: "0x0",
      extra: {
        addedOwners: [],
        removedOwners: []
      },
      sig: "0x68affbe7c2ad15fdec05427fc7fa94cb885e28525463ab2b04207b5972c6bb5e57c9596f98b5e1e5c8896e46b643a8734cfd30934272012c90081209dc15a0f300"
  }, {
      blockchainID: "0x69eb463bc4f6b2df",
      blockhash: "0x6d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4",
      blocknumber: "0x1",
      extra: {
        addedOwners: [...],
        removedOwners: []
      },
      sig: "0x5a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201"
  }]
}
```
```
# Creates Layer3 Blockchain (AnchorTx with blocknumber #0)
curl -X POST --data  '{"jsonrpc":"2.0","id":"14","method":"plasma_sendAnchorTransaction","params":[{"blockchainID":"0x69eb463bc4f6b2df", "blocknumber":"0x0", "blockhash":"0x03c85f1da84d9c6313e0c34bcb5ace945a9b12105988895252b88ce5b769f82b", "sig":"0x68affbe7c2ad15fdec05427fc7fa94cb885e28525463ab2b04207b5972c6bb5e57c9596f98b5e1e5c8896e46b643a8734cfd30934272012c90081209dc15a0f300"}]}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "14",
    "result": "0x6808da8621d7c01021dbd98c37344e48418dd1967d6302d73a7b1c9341ca3be1"
}


# Add/Remove Owners for a Layer3 Blockchain (AnchorTx with non-empty extradata)
curl -X POST --data '{"jsonrpc":"2.0","id":"15","method":"plasma_sendAnchorTransaction","params":[{"blockchainID":"0x69eb463bc4f6b2df", "blocknumber":"0x1", "blockhash":"0x6d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4", "extra":{"addedOwners":["0x3088666E05794d2498D9d98326c1b426c9950767"],"removedOwners":[]}, "sig":"0x5a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201"}]}' -H'content-type:application/json;' 'localhost:8505'
{
    "jsonrpc": "2.0",
    "id": "15",
    "result": "0xb6f91e2a3059e7a2d347d76d2e548f0a3c58068f0185f5dadcceb9580591657a"
}

# Retrieve Pending AnchorTxs from AnchorTransactionPool
curl -X POST --data  '{"jsonrpc":"2.0","id":"16","method":"plasma_getAnchorTransactionPool"}' -H 'content-type:application/json;' 'localhost:8505';
{
    "jsonrpc": "2.0",
    "id": "16",
    "result": {
        "7632271216030954207": [
            {
                "blockchainID": "0x69eb463bc4f6b2df",
                "blocknumber": "0x0",
                "blockhash": "0x03c85f1da84d9c6313e0c34bcb5ace945a9b12105988895252b88ce5b769f82b",
                "extra": {
                    "addedOwners": [],
                    "removedOwners": []
                },
                "sig": "0x68affbe7c2ad15fdec05427fc7fa94cb885e28525463ab2b04207b5972c6bb5e57c9596f98b5e1e5c8896e46b643a8734cfd30934272012c90081209dc15a0f300"
            },
            {
                "blockchainID": "0x69eb463bc4f6b2df",
                "blocknumber": "0x1",
                "blockhash": "0x6d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4",
                "extra": {
                    "addedOwners": [
                        "0x3088666e05794d2498d9d98326c1b426c9950767"
                    ],
                    "removedOwners": []
                },
                "sig": "0x5a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201"
            }
        ]
    }
}
```


## Transaction Signing
Both Plasma Transaction and Anchor Transaction are signed as *ethereum message*, which are prepended with `\x19Ethereum Signed Message:\n32`. Use [PlasmaTx](https://rinkeby.etherscan.io/address/0x36150061b09da5304cc4ca6fe3a1c9888bd6561a#readContract) and [AnchorTx](https://rinkeby.etherscan.io/address/0x0037a175040810d45b94373152ee6660f57ee40d#readContract) debug tool to verify that signing is correct.

```
# PlasmaTx Signing Example

PlasmaTx To Sign:
{
    "tokenID": 0x37b01bd3adfc4ef3,
    "denomination": 1000000000000000000,
    "depositIndex": 1,
    "prevBlock": 1,
    "prevOwner": "0x82Da88C31E874C678D529ad51E43De3A4BAF3914",
    "recipient": "0x3088666E05794d2498D9d98326c1b426c9950767",
    "allowance": 100000000000000001,
    "spent": 200000000000000002,
    "sig": "" //unknown yet
}

# Step 1: unsignedTxbyte
rlp.encode([37b01bd3adfc4ef3,de0b6b3a7640000, 1, 1, 82da88c31e874c678d529ad51e43de3a4baf3914, 3088666e05794d2498d9d98326c1b426c9950767, 16345785d8a0001, 2c68af0bb140002, ""])
0xf8518837b01bd3adfc4ef3880de0b6b3a764000001019482da88c31e874c678d529ad51e43de3a4baf3914943088666e05794d2498d9d98326c1b426c995076788016345785d8a00018802c68af0bb14000280 // result of rlpencode(PlasmaTx)

# Step 2: Retrieve shortHash
0x81bc3f429f14473e759729c145c73b20f00006b8be6902f6a47326c750051f3a // Keccak256(unsignedTxbyte)

# step 3: Sign with Metamask (web3.eth.sign)
signedHash: 0x81bc3f429f14473e759729c145c73b20f00006b8be6902f6a47326c750051f3a // Keccak256("\x19Ethereum Signed Message:\n32", shortHash)
signature: 0xffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600

# Step 4: Fill in the signature
{"tokenID":0x37b01bd3adfc4ef3, "denomination": 1000000000000000000, "depositIndex": 1, "prevBlock": 1, "prevOwner": "0x82Da88C31E874C678D529ad51E43De3A4BAF3914", "recipient": "0x3088666E05794d2498D9d98326c1b426c9950767", "allowance": 100000000000000001, "spent": 200000000000000002, "sig": "0xffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600"}

# Step 5: Send as RawTransaction
txBytes:
0xf8938837b01bd3adfc4ef3880de0b6b3a764000001019482da88c31e874c678d529ad51e43de3a4baf3914943088666e05794d2498d9d98326c1b426c995076788016345785d8a00018802c68af0bb140002b841ffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600 // rlpencode(signedPlasmaTx)

> plasma.sendRawPlasmaTransaction("0xf8938837b01bd3adfc4ef3880de0b6b3a764000001019482da88c31e874c678d529ad51e43de3a4baf3914943088666e05794d2498d9d98326c1b426c995076788016345785d8a00018802c68af0bb140002b841ffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600")
"0x8dde5209f1c65c0dbfd88a75ea31160823184b410984765f0a4624e649215df1" // Keccak256(txBytes)

rlpDecode:

[
  37b01bd3adfc4ef3,
  0de0b6b3a7640000,
  01,
  01,
  82da88c31e874c678d529ad51e43de3a4baf3914,
  3088666e05794d2498d9d98326c1b426c9950767,
  016345785d8a0001,
  02c68af0bb140002,
  ffe9cddfd9306418f8b7dc6192c3abb1b7a4ef119d819219e9b7f5bcb5e4c02a71cd4a4d73e2a0dbb359d76867e448060f214f7a1cdd8ecade0f3e7684cf0a1600,
]

Done!
```

```
# AnchorTx Signing Example

AnchorTX To Sign:
{
    "blockchainID": "0x69eb463bc4f6b2df",
    "blocknumber": "0x1",
    "blockhash": "0x6d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4",
    "extra": {
        "addedOwners": [
            "0x3088666E05794d2498D9d98326c1b426c9950767"
        ],
        "removedOwners": []
    },
    "sig": "" // unknown yet
}

# Step 1: unsignedTxbyte
RLPENCODE([69eb463bc4f6b2df, 1, 0x6d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4,[[3088666e05794d2498d9d98326c1b426c9950767],[]], ""])
0xf8448869eb463bc4f6b2df01a06d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4d7d5943088666e05794d2498d9d98326c1b426c9950767c080 // result of rlpencode(AnchorTX)

# Step 2: Retrieve shortHash
0x4c0b719111a7a240647d5a3f266842f97333e67bf2c493d45e7e27a52660f801 // Keccak256(unsignedTxbyte)

# step 3: Sign with Metamask (web3.eth.sign)
signedHash: 0x97eae61b4aa00dc438f9df18a359d1ce0dd8d0d0be900440bcbd18f01c8a9509 // Keccak256("\x19Ethereum Signed Message:\n32", shortHash)
signature: 0x5a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201

# Step 4: Fill in the signature
{"BlockChainID":"69eb463bc4f6b2df", "BlockNumber":"1", "BlockHash":"6d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4", "Extra":"d7d5943088666e05794d2498d9d98326c1b426c9950767c0", "Sig":"5a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201"}

# Step 5: Send as RawTransaction
txBytes:
0xf8868869eb463bc4f6b2df01a06d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4d7d5943088666e05794d2498d9d98326c1b426c9950767c0b8415a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201 // rlpencode(signedAnchorTx)

> plasma.sendRawAnchorTransaction("0xf8868869eb463bc4f6b2df01a06d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4d7d5943088666e05794d2498d9d98326c1b426c9950767c0b8415a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201")
"0xb6f91e2a3059e7a2d347d76d2e548f0a3c58068f0185f5dadcceb9580591657a" // Keccak256(txBytes)

rlpDecode:

[
  69eb463bc4f6b2df,
  01,
  6d255fc3390ee6b41191da315958b7d6a1e5b17904cc7683558f98acc57977b4,
  [
    [
      3088666e05794d2498d9d98326c1b426c9950767,
    ],
    [],
  ],
  5a09945a81e39455acc1c3c97b559112ec9e874c0964c38f3c119efc10d3a663433e0726869579edf2e2285f676a6cb1330b6e22dd6a19398b630002841ed08201,
]

Done!
```

## Contract Interface

* [PlasmaTx Debug Tool](https://rinkeby.etherscan.io/address/0x36150061b09da5304cc4ca6fe3a1c9888bd6561a#readContract): The library handles PlasmaTx RLP encode/decode and verifies a PlasmaTx's validity by checking (1) recipient matches with depositor when prevBlock is 0 or (2) signer matches with prevOwner otherwise. Additional helper functions have been provided in this contract but not used in production code.  

* [AnchorTx Debug Tool](https://rinkeby.etherscan.io/address/0x0037a175040810d45b94373152ee6660f57ee40d#readContract): The library handles AnchorTx and Extra struct RLP encode/decode on chain. Additional helper functions have been provided in this contract but not used in production code.

* [SparseMerkle Tree](https://rinkeby.etherscan.io/address/0xda4d188831b6c67140cefec35f540bacd87ba526#readContract): This contract is implemented based on [succinct sparse merkle proof format](https://ethresear.ch/t/plasma-cash-with-sparse-merkle-trees-bloom-filters-and-probabilistic-transfers) we published on ethresearch.

* [RootContract](https://rinkeby.etherscan.io/address/0x97f33d99d6d473cb938abb893abd49a2bb1404bf): Link to our feature-complete plasma contract on Rinkeby


## Events on MainNet: Deposits, Exits, and Challenges

We define plasma nodes as a set of nodes that are responsible of (1) monitoring events {`Deposits`, `Exits`, `Challenges`} happening on RootChain contract and recording those state changes on Plasma (2) accepting RPC requests for tokenID transfers on Plasma Chain. Among all nodes, one "main" node and multiple "stand-by" nodes are strategically selected as "Leader" group with additional responsibilities of publishing block info {Sparse Merkle Root, BloomFilter Hash} periodically on the RootChain contract.

Generally speaking, plasma nodes are required to be online at all time and watch any incoming RootChain event via plasma.eventHandler; if a plasma node temporarily went offline and wants to rejoin the network, it must catch up to the latest state on MainNet by retrieving all the missing past events via RPC calls with filter options.  A list of event classes that need responding to are:

* `PublishedBlock`: events generated by Plasma Leader calling `submitBlock(blockNumber, merkleRoot Hash)`. Every PublishedBlock event includes the `currentDepositIndex` at given blockNumber, which can be used by plasma nodes to filter a list of unprocessed deposits on plasma chain.  

* `Deposit`: events generated by users depositing ETH(WLK) into RootChain contract. Plasma nodes call `initDeposit()` on plasma chain and sign-off valid tokenIDs to the depositors. Periodically, a plasma block containing all RootChain `deposit()` and all token transfers occurred on plasma chain _*since last published block*_ should be minted, resulting in a new `submitBlock` call by the Plasma Leader.

* `StartExit` and `DepositExit`: events generated by users trying to withdraw tokenIDs they claim to have ownership for. Malicious/frivolous exits can be immediately challenged by the anyone, while valid exits should result in a plasma node subsequently responding with the tokenID as being _untransferrable_ in a `sendPlasmaTransaction` call.

* `ChallengeExit`: ChallengeExit currently _does not_ have any event. A valid challenge remove the tokenID from pending exitsQueue made by illegitimate owner; restrictions on such token be immediately lifted. e.g. marking the tokenID _transferrable_ in a `sendPlasmaTransaction` call.  

* `FinalizeExit`: all tokens finalized can be taken out of circulation completely; the plasma node should respond with the tokenID as being _exited_

## Simulations

* [Vanilla case](https://gist.github.com/mkchungs/d06e325408d503795f39e96e716752be): A 100 block, 25ms per block sample simulation log can be found here. "Smart Exit Proof" is automatically generated by operator and returned by JSON-RPC call

* [FaultyTx](https://github.com/wolkdb/deepblockchains/pull/2): Fixed. Simulation not available.

* Reorg Case: TBA


## Resources

* [Go-Plasma](https://github.com/wolkdb/plasma): plasma implementation in Go (*pending for security audit*)

* sql Chain : TBA

* nosql Chain : TBA

For technical support, please contact michael@wolk.com
