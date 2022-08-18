#  Spam-protected chat2 application with on-chain group management

This document is a tutorial on how to run the chat2 application in the spam-protected mode using the Waku-RLN-Relay protocol and with dynamic/on-chain group management.
In the on-chain/dynamic group management, the state of the group members i.e., their identity commitment keys is moderated via a membership smart contract deployed on the Goerli network which is one of the Ethereum testnets.
Members can be dynamically added to the group and the group size can grow up to 2^20 members.
This  differs from the prior test scenarios in which the RLN group was static and the set of members' keys was hardcoded and fixed.


## Prerequisites 
To complete this tutorial, you will need 1) an account with at least `0.001` ethers on the Goerli testnet and 2) a hosted node on the Goerli testnet. 
In case you are not familiar with either of these two steps, you may follow the following tutorial to fulfill the [prerequisites of running on-chain spam-protected chat2](./pre-requisites-of-running-on-chain-spam-protected-chat2.md).
Note that the required `0.001` ethers correspond to the registration fee, 
however, you still need to have more funds in your account to cover the cost of the transaction gas fee.

## Overview
Figure 1 provides an overview of the interaction of the chat2 clients with the test fleets and the membership contract. 
At a high level,  when a chat2 client is run with Waku-RLN-Relay mounted in on-chain mode, it creates RLN credential (i.e., an identity key and an identity commitment key) and 
sends a transaction to the membership contract to register the corresponding membership identity commitment key.
This transaction will also transfer `0.001` Ethers to the contract as membership fee.
This amount plus the transaction fee will be deducted from the supplied Goerli account. 
Once the transaction is mined and the registration is successful, the registered credential will get displayed on the console of your chat2 client.
You may copy the displayed RLN credential and reuse them for the future execution of the chat2 application.
Proper instructions in this regard is provided in the following [section](#how-to-reuse-rln-credential).
If you choose not to reuse the same credential, then for each execution, a new registration will take place and more funds will get deducted from your Goerli account.
Under the hood, the chat2 client constantly listens to the membership contract and keeps itself updated with the latest state of the group.

In the following test setting, the chat2 clients are to be connected to the Waku test fleets as their first hop. 
The test fleets will act as routers and are also set to run Waku-RLN-Relay over the same pubsub topic and content topic as chat2 clients i.e., the default pubsub topic of `/waku/2/default-waku/proto` and the content topic of `/toy-chat/2/luzhou/proto`. 
Spam messages published on the said combination of topics will be caught by the test fleet nodes and will not be routed.
Note that spam protection does not rely on the presence of the test fleets.
In fact, all the chat2 clients are also capable of catching and dropping spam messages if they receive any.
You can test it by connecting two chat2 clients (running Waku-RLN-Relay) directly to each others and see they can spot each others' spam activities.

 ![](./imgs/rln-relay-chat2-overview.png)
 Figure 1.

# Set up

## Prerequisites

1. Install Go language, by following the instructions from https://go.dev/doc/install and a C compiler (If using a debian based distro, you can run `apt install build-essential`)

2. Clone the repository
```
git clone https://github.com/status-im/go-waku
cd go-waku
```

## Build chat2
```
make chat2
```

## Set up a chat2 client

Run the following command to set up your chat2 client. 

```
./build/chat2 --fleet=test --content-topic=/toy-chat/2/luzhou/proto --rln-relay=true --rln-relay-dynamic=true --eth-mem-contract-address=0x4252105670fe33d2947e8ead304969849e64f2a6 --eth-account-privatekey=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx --eth-client-address=xxxx  
```

In this command
- the `--fleet:test` indicates that the chat2 app gets connected to the test fleets.
- the `toy-chat/2/luzhou/proto` passed to the `content-topic` option indicates the content topic on which the chat2 application is going to run.
- the `rln-relay` flag is set to `true` to enable the Waku-RLN-Relay protocol for spam protection.
- the `--rln-relay-dynamic` flag is set to `true`  to enable the on-chain mode of  Waku-RLN-Relay protocol with dynamic group management.
- the `--eth-mem-contract-address` option gets the address of the membership contract.
  The current address of the contract is `0x4252105670fe33d2947e8ead304969849e64f2a6`.
  You may check the state of the contract on the [Goerli testnet](https://goerli.etherscan.io/address/0x4252105670fe33d2947e8ead304969849e64f2a6).
- the `eth-account-privatekey` option is for your account private key on the Goerli testnet. 
  It is made up of 64 hex characters (not sensitive to the `0x` prefix).
- the `eth-client-address` should be assigned with the address of a websocket endpoint of the hosted node on the Goerli testnet. 
  You need to replace the `xxxx` with the actual node's websocket endpoint.

For the last two config options i.e., `eth-account-privatekey`, and `eth-client-address`,  if you do not know how to obtain those, you may use the following tutorial on the [prerequisites of running on-chain spam-protected chat2](https://github.com/status-im/nwaku/blob/master/docs/tutorial/pre-requisites-of-running-on-chain-spam-protected-chat2.md).

> You may set up more than one chat client, using the `--rln-relay-membership-credentials-file` flag, specifying in each client a different path to store the credentials.

Once you run the command, you will see the following message:
```
Setting up dynamic rln...
```
At this phase, your RLN credential are getting created and a transaction is being sent to the membership smart contract.
It will take some time for the transaction to be finalized. Afterwards, messages related to setting up the connections of your chat app will be shown,
the content may differ on your screen though:
```
INFO: Welcome, Anonymous!
INFO: type /help to see available commands 

INFO Listening on 
  - /ip4/75.157.120.249/tcp/60001/p2p/16Uiu2HAmQXuZmbjFWGagthwVsPFrc5ZrZ9c53qdUA45TWoZaokQn
```
The registered RLN identity key, the RLN identity commitment key, and the index of the registered credential will be displayed as given below.
The RLN identity key is not shown in the figure (replaced by a string of `x`s) for security reasons. But, you will see your RLN identity key.
```
INFO: RLN config:
- Your membership index is: 63
- Your RLN identity key is: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
- Your RLN identity commitment key is: 6c6598126ba10d1b70100893b76d7f8d7343eeb8f5ecfd48371b421c5aa6f012

INFO: attempting DNS discovery with enrtree://ANTL4SLG2COUILKAPE7EF2BYNL2SHSHVCHLRD5J7ZJLN5R3PRJD2Y@prod.waku.nodes.status.im
INFO: Discovered and connecting to [/dns4/node-01.gc-us-central1-a.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAmVkKntsECaYfefR1V2yCR79CegLATuTPE6B9TxgxBiiiA /dns4/node-01.ac-cn-hongkong- c.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD /dns4/node-01.do-ams3.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAmL5okWopX7NqZWBUKVqW8iUxCEmd5GMHLVPwCgzYzQv3e]
INFO: Connected to 16Uiu2HAmVkKntsECaYfefR1V2yCR79CegLATuTPE6B9TxgxBiiiA
INFO: Connected to 16Uiu2HAmL5okWopX7NqZWBUKVqW8iUxCEmd5GMHLVPwCgzYzQv3e
INFO: Connected to 16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD
INFO: No store node configured. Choosing one at random...
```
You will also see some historical messages  being fetched, again the content may be different on your end:

```
[Jul 26 10:41 Bob] hi
[Jul 26 10:41 Bob] hi
[Jun 29 16:21 Alice] spam1
[Jun 29 16:21 Alice] hiiii
[Jun 29 16:21 Alice] hello
[Jun 29 16:19 Bob] hi
[Jun 29 16:19 Bob] hi
[Jun 29 16:19 Alice] hi
[Jun 29 16:15 b] hi
[Jun 29 16:15 h] hi
...
```

Finally, the chat prompt `Send a message...` will appear which means your chat2 client is ready.
Once you type a chat line and hit enter, you will see a message that indicates the epoch at which the message is sent e.g.,

```
INFO: RLN Epoch: 165886530

[Jul 26 12:55 Anonymous] Hi
```
The numerical value `165886530` indicates the epoch of the message `Hi`.
You will see a different value than `165886530` on your screen. 
If two messages sent by the same chat2 client happen to have the same RLN epoch value, then one of them will be detected as spam and won't be routed (by test fleets in this test setting).
You'll also see a `ERROR: validation failed` message
At the time of this tutorial, the epoch duration is set to `10` seconds.
You can inspect the current epoch value by checking the following [constant variable](https://github.com/status-im/go-rln/blob/main/rln/types.go#L194) in the go-rln codebase.
Thus, if you send two messages less than `10` seconds apart, they are likely to get the same `rln epoch` values.

After sending a chat message, you may experience some delay before the next chat prompt appears. 
The reason is that under the hood a zero-knowledge proof is being generated and attached to your message.

Try to spam the network by violating the message rate limit i.e.,
sending more than one message per epoch. 
Your messages will be routed via test fleets that are running in spam-protected mode over the same content topic i.e., `/toy-chat/2/luzhou/proto` as your chat client.
Your spam activity will be detected by them and your message will not reach the rest of the chat clients.
You can check this by running a second chat user and verifying that spam messages are not displayed as they are filtered by the test fleets.
A sample test scenario is illustrated in the [Sample test output section](#sample-test-output).

Once you are done with the test, make sure you close all the chat2 clients by typing the `/exit` command.
```
>> /exit
Bye!
```

## How to reuse RLN credential

You may reuse your old RLN credential using `rln-relay-membership-index`, `rln-relay-id` and `rln-relay-id-commitment` options. 
For instance, if the previously generated credential are
```
INFO: RLN config:
- Your membership index is: 63
- Your rln identity key is: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
- Your rln identity commitment key is: 6c6598126ba10d1b70100893b76d7f8d7343eeb8f5ecfd48371b421c5aa6f012
```
Then, the execution command will look like this (inspect the last three config options):
```
./build/chat2 --fleet=test --content-topic=/toy-chat/2/luzhou/proto --rln-relay=true --rln-relay-dynamic=true --eth-mem-contract-address=0x4252105670fe33d2947e8ead304969849e64f2a6 --eth-account-privatekey=your_eth_private_key  --eth-client-address=your_goerli_node --rln-relay-membership-index=63 --rln-relay-id=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx --rln-relay-id-commitment:6c6598126ba10d1b70100893b76d7f8d7343eeb8f5ecfd48371b421c5aa6f012

```

# Sample test output
In this section, a sample test of running two chat clients is provided.
Note that the values used for `eth-account-privatekey`, and `eth-client-address` in the following code snippets are junk and not valid.

The two chat clients namely `Alice` and `Bob` are connected to the test fleets.
`Alice` sends 4 messages i.e., `message1`, `message2`, `message3`, and `message4`.
However, only three of them reach `Bob`. 
This is because the two messages `message2` and `message3` have identical RLN epoch values, so, one of them gets discarded by the test fleets as a spam message. 
The test fleets do not relay `message3` further, hence `Bob` never receives it.
You can check this fact by looking at `Bob`'s console, where `message3` is missing. 

**Alice**
``` 
./build/chat2 --fleet=test --content-topic=/toy-chat/2/luzhou/proto --rln-relay=true --rln-relay-dynamic=true --eth-mem-contract-address=0x4252105670fe33d2947e8ead304969849e64f2a6 --eth-account-privatekey=your_eth_private_key --eth-client-address=your_goerli_node --rln-relay-membership-credentials-file=rlnCredentialsAlice.txt --nickname=Alice

Seting up dynamic rln
INFO: Welcome, Alice!
INFO: type /help to see available commands

INFO Listening on 
  - /ip4/75.157.120.249/tcp/60001/p2p/16Uiu2HAmQXuZmbjFWGagthwVsPFrc5ZrZ9c53qdUA45TWoZaokQn

INFO: RLN config:
  - Your membership index is: 64
  - Your rln identity key is: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
  - Your rln identity commitment key is: bd093cbf14fb933d53f596c33f98b3df83b7e9f7a1906cf4355fac712077cb28

INFO: attempting DNS discovery with enrtree://ANTL4SLG2COUILKAPE7EF2BYNL2SHSHVCHLRD5J7ZJLN5R3PRJD2Y@prod.waku.nodes.status.im
INFO: Discovered and connecting to [/dns4/node-01.gc-us-central1-a.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAmVkKntsECaYfefR1V2yCR79CegLATuTPE6B9TxgxBiiiA /dns4/node-01.ac-cn-hongkong- c.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD /dns4/node-01.do-ams3.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAmL5okWopX7NqZWBUKVqW8iUxCEmd5GMHLVPwCgzYzQv3e]
INFO: Connected to 16Uiu2HAmVkKntsECaYfefR1V2yCR79CegLATuTPE6B9TxgxBiiiA
INFO: Connected to 16Uiu2HAmL5okWopX7NqZWBUKVqW8iUxCEmd5GMHLVPwCgzYzQv3e
INFO: Connected to 16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD
INFO: No store node configured. Choosing one at random...

>> message1

INFO RLN Epoch: 165886591

[Jul 26 13:05 Alice] message1

>> message2

INFO RLN Epoch: 165886592

[Jul 26 13:05 Alice] message2

>> message3

INFO RLN Epoch: 165886592
ERROR: validation failed

>> message4

INFO RLN Epoch: 165886593

[Jul 26 13:05 Alice] message4
>> 
```

**Bob**
``` 
./build/chat2 --fleet=test --content-topic=/toy-chat/2/luzhou/proto --rln-relay=true --rln-relay-dynamic=true --eth-mem-contract-address=0x4252105670fe33d2947e8ead304969849e64f2a6 --eth-account-privatekey=your_eth_private_key --eth-client-address=your_goerli_node --rln-relay-membership-credentials-file=rlnCredentialsBob.txt --nickname=Bob

Seting up dynamic rln
INFO: Welcome, Bob!
INFO: type /help to see available commands

INFO Listening on 
  - /ip4/75.157.120.249/tcp/60001/p2p/16Uiu2HAmQXuZmbjFWGagthwVsPFrc5ZrZ9c53qdUA45TWoZaokQn

INFO: RLN config:
  - Your membership index is: 65
  - Your rln identity key is: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
  - Your rln identity commitment key is: bd093cbf14fb933d53f596c33f98b3df83b7e9f7a1906cf4355fac712077cb28

INFO: attempting DNS discovery with enrtree://ANTL4SLG2COUILKAPE7EF2BYNL2SHSHVCHLRD5J7ZJLN5R3PRJD2Y@prod.waku.nodes.status.im
INFO: Discovered and connecting to [/dns4/node-01.gc-us-central1-a.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAmVkKntsECaYfefR1V2yCR79CegLATuTPE6B9TxgxBiiiA /dns4/node-01.ac-cn-hongkong- c.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD /dns4/node-01.do-ams3.wakuv2.prod.statusim.net/tcp/8000/wss/p2p/16Uiu2HAmL5okWopX7NqZWBUKVqW8iUxCEmd5GMHLVPwCgzYzQv3e]
INFO: Connected to 16Uiu2HAmVkKntsECaYfefR1V2yCR79CegLATuTPE6B9TxgxBiiiA
INFO: Connected to 16Uiu2HAmL5okWopX7NqZWBUKVqW8iUxCEmd5GMHLVPwCgzYzQv3e
INFO: Connected to 16Uiu2HAm4v86W3bmT1BiH6oSPzcsSr24iDQpSN5Qa992BCjjwgrD
INFO: No store node configured. Choosing one at random...

[Jul 26 13:05 Alice] message1
[Jul 26 13:05 Alice] message2
[Jul 26 13:05 Alice] message4
```