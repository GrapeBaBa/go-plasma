#!/bin/bash
echo 'Publish block'

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd07247671e5902ab9767b5915ea3402084280e689505e037897523e7bca6e90e733c0000000000000000000000000000000000000000000000000000000000000001", "value": "0x0", "gas": "0x186a0"}],"id":6}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "   #1"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd072010c516c398e8db4f2d997c577c4a077a4b910531f9d8bb4a8e1ed4e8b38c4f30000000000000000000000000000000000000000000000000000000000000002", "value": "0x0", "gas": "0x186a0"}],"id":7}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "   #2"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd072fb713fa1d1ef39695938c8364e1eb1587d7444c5a67214e7d77e690ed9b930a20000000000000000000000000000000000000000000000000000000000000003", "value": "0x0", "gas": "0x186a0"}],"id":8}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "   #3"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd072a719c0d31ee1f17809a9737ca065274ef6446536d2609e49c96719d165ec52200000000000000000000000000000000000000000000000000000000000000004", "value": "0x0", "gas": "0x186a0"}],"id":9}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "   #4"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd0728ac99bea1414060e6316265751de45379cc9312ef2849ceeec3e663269f27cf70000000000000000000000000000000000000000000000000000000000000005", "value": "0x0", "gas": "0x186a0"}],"id":10}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "  #5"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd07211ad945132dd5f7d87a49716d2481ceed4bfdac0b1954c9f3d368685b85e1cfd0000000000000000000000000000000000000000000000000000000000000006", "value": "0x0", "gas": "0x186a0"}],"id":11}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "  #6"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd072f76941d68904f50ce3b3f71313d4f47fd556e49b9a2328df855c2aaad13640c90000000000000000000000000000000000000000000000000000000000000007", "value": "0x0", "gas": "0x186a0"}],"id":12}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "  #7"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd07223c64ee6366fa796d568f4ae4a55d5d780efbd02b00ea23abc2346e51010b18c0000000000000000000000000000000000000000000000000000000000000008", "value": "0x0", "gas": "0x186a0"}],"id":13}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "  #8"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd072941fe3be9a48b7fa11daccd5be9440981df6a7f1d194c4fd2d91c2c1042b2f3b0000000000000000000000000000000000000000000000000000000000000009", "value": "0x0", "gas": "0x186a0"}],"id":14}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "  #9"

curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xefcfd072142baa422d6ac8e9a729e9f9f449e5f1b803a85951308ae56e6d7c20b7885bb2000000000000000000000000000000000000000000000000000000000000000a", "value": "0x0", "gas": "0x1f6a0"}],"id":15}' -H "Content-Type: application/json" -X POST localhost:8545;
echo "  #10"

echo
