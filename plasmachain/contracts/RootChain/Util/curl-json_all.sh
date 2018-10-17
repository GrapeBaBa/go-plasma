#!/bin/bash

##echo 'Step 0: ganache backend'
##ganache-cli  -v -m "elephant obey fiction shift punch devote glue recipe sunset sock tube obtain" -e 1000 -i 4 -h "127.0.0.1" -p "8545"
#echo

##echo 'Step 0.1: javascript event listener'
##node eventcrawler.js writeLog events.log

echo 'Step 1: init Contract'
sh curl-json_init.sh
echo

echo 'Step 2: Deposit'
sh curl-json_deposit.sh
echo

echo 'Step 3: Sumitblocks'
sh curl-json_publishblock.sh
echo

echo 'Step 4: Exit and Challenge'
sh curl-json_exitchallenge.sh
echo
