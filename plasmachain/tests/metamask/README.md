
# RLP Encoding of App Transaction

```$ go test -run Metamask
AppTx: {"ts":"1529000648", "data":"0abcdef123456789"}
Encoded bytes: ce845b22b2c8880abcdef123456789
Decoded Tx: {"ts":"1529000648", "data":"0abcdef123456789"}
msgHash: 39867be5f9b67a02f1f9cad5784e28557c614c8024b351e6cad5f9119e268309
sig: 87eb152bea5d3ca68d5127a988ac3a0ba053e418b60679c6d84a6ff96746ebe91ef691e3dfc7a903b128f4f365a5debac9282943d516f96fb84219029b462c9e1b
Addr: edafcc405196a51ae462442e870d0ea9d5d395a1 is correct!
PASS
ok  	github.com/wolkdb/go-plasma/plasmachain/tests/metamask	0.026s
```

You can check decoding of this with `rlp decode` on www6001 (`5b22b2c8` is `1529000648` and data matches up):
```
$ rlp decode 0xce845b22b2c8880abcdef123456789
[ '5b22b2c8', '0abcdef123456789' ]
```
