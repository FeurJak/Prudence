# Prudence Labs

## Packages: 

### ABI: 

Just a stripped down geth ABI pkg, so you can easily interface with Contract functions. 

### GETH_CLI: 

RPC interface for GETH node. 

### Precog: 

General backend for connecting to the P2P network (i.e. geth peers, mempool txs) and remote state access (either via IPC / RPC or disk access). 

### SVE 
Sustainable Virtual Economies (SVE). Don't ask me about it.

## Lab 1 / 1 / 2024

Test P2P Network. 
 - Find & Connect to Peers
 - Build mempool from Peer Set.
 - Test mempool coverage (% of block Txs seen in mempool).

## Lab 2 / 1 / 2024

Integrate EVM To Prudence Labs. 
- Demonstrate ability to hook EVM to remote state access.

Migrate SVE pkgs to Prudence Labs. 
- Demonstrate ability to fetch ERC20 Token Metadata (i.e. Max Tx / Wallet, buy / sell tax)
