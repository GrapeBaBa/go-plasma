echo "\n\nDeposit #0 [0xb437230feb2d24db]"
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xd0e30db0", "value": "0xde0b6b3a7640000", "gas": "0x13880"}],"id":2}' -H "Content-Type: application/json" -X POST localhost:8545;

echo "\n\nDeposit #1 [0x37b01bd3adfc4ef3]"
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0x82Da88C31E874C678D529ad51E43De3A4BAF3914","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xd0e30db0", "value": "0xde0b6b3a7640000", "gas": "0x13880"}],"id":3}' -H "Content-Type: application/json" -X POST localhost:8545;

echo "\n\nDeposit #2 [0xb76883d225414136]"
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0x3088666E05794d2498D9d98326c1b426c9950767","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xd0e30db0", "value": "0x1bc16d674ec80000", "gas": "0x13880"}],"id":4}' -H "Content-Type: application/json" -X POST localhost:8545;

echo "\n\nDeposit #3 [0x9af84bc1208918b]"
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xBef06CC63C8f81128c26efeDD461A9124298092b","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xd0e30db0", "value": "0x29a2241af62c0000", "gas": "0x13880"}],"id":5}' -H "Content-Type: application/json" -X POST localhost:8545;

echo "\n\nDeposit #4 [0x7c00dfa72e8832ed]"
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0x74f978A3E049688777E6120D293F24348BDe5fA6","to": "0xa611dd32bb2cc893bc57693bfa423c52658367ca", "data": "0xd0e30db0", "value": "0x3782dace9d900000", "gas": "0x13880"}],"id":6}' -H "Content-Type: application/json" -X POST localhost:8545;

echo
