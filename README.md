# True Nodes - Phantom Daemon

[![Total Downloads](https://img.shields.io/github/downloads/TrueNodes/phantom/total.svg)](https://github.com/TrueNodes/phantom/releases)
[![Last Version](https://img.shields.io/github/release/TrueNodes/phantom/all.svg)](https://github.com/TrueNodes/phantom/releases)
[![Last Release Date](https://img.shields.io/github/release-date/TrueNodes/phantom.svg)](https://github.com/TrueNodes/phantom/releases/latest)
![Lines of Code](https://img.shields.io/tokei/lines/github/TrueNodes/phantom.svg)
[![GitHub Stars](https://img.shields.io/github/stars/TrueNodes/phantom.svg)](https://github.com/TrueNodes/phantom/stargazers)
[![GitHub Forks](https://img.shields.io/github/forks/TrueNodes/phantom.svg)](https://github.com/TrueNodes/phantom/network)

---

Want to host your masternode easily and at low cost, without worrying about programming? **True Nodes - Masternode Hosting** Contact me on [Telegram](https://t.me/matheus_bach). (please send an objective message right away so I don't confuse you with spam)

---

Phantom nodes requires no static IP address, no copy of the blockchain, and no proof-of-service. As such, you can run a node on any IP address of your liking: `1.1.1.1` or `8.8.8.8` if you wish. Phantoms also support live hot-swap with currently running nodes, there is no need to re-queue.

The phantom daemon is extremely lightweight allowing you to run hundreds of nodes from a modest machine if you wished. And, possibly most importantly, you can move your currently running masternodes to phantom nodes without restarting since a real IP address is no longer a requirement.

The phantom daemon is custom built wallet designed to replicate only what is required for pre-EVO masternodes to run; it replaces the masternode daemon piece. It does not handle any wallet private keys and has no access to your coins. You will still need a wallet to start your masternodes, but once started, the phatom node system will handle the rest for you.

TrueNodes Phantom
Original Dev breakcrypto, contributions by carsenk and universecreditcoin. This fork is maintained by matheusbach and TrueNodes

## Contact information

> telegram: [@matheus_bach](https://t.me/matheus_bach)    
> discord: [PeixeLua#4822](https://discord.com/users/PeixeLua#4822)    

> email: admin@denarius.io    
> twitter: https://twitter.com/carsenjk   
> discord: Carsen#3333    
> Discord Chat: https://discord.gg/UPpQy3n    

> email: breakcrypto@gmail.com    
> twitter: https://twitter.com/_breakcrypto   
> discord: breakcrypto#0011   
> discord channel: https://discord.gg/fQPb2ew   
> bitcoin talk discussion: https://bitcointalk.org/index.php?topic=5136453.0    

# Feature

* Fully self-sufficient
* Minimal memory and disk usage (in the 10s of megabytes vs. gigabytes)
* Hot-swap with live node daemons, no restart required
* Select any IP address, no static IP required
* Auto-load settings from a coinconf.json 
* Optionally auto-load bootstrap hashes and peers from Iquidus explorers (more APIs coming soon)
* Epoch timestamp support for high-availability deterministic pings
* Use your existing masternode.conf if you don't want deterministic pings
* Runs on windows, linux, mac, arm, and more.

# A note from the original developer

Phantoms have been released to make it easier, and less costly, for masternode supporters to host their own nodes. Masternode hosting companies are free to utilize the phantom system as long as they comply with the terms of the Server Side Public License. 

# Quick start

Download a binary release from below. See if there's a coin configuration for the coin you're wishing to use. If not, you'll need to locate the proper settings. There are notes below on where to look or feel free to ask on discord, reddit, or btct. If there is a coin conf for your coin then switching over to phantoms is easy:

```
./phantom -coin_conf="/path/to/coin.conf" -masternode_conf="/path/to/masternode.conf"
```

That's it. You do not need to restart your masternodes, you don't need to change IP addresses, etc. Once the phantom daemon is running, you can disable your masternode daemons, cancel most of VPS subscriptions, and enjoy the savings. You'll know the phantoms are working when you see the active time refresh (can take up to 20 minutes). If that active time doesn't update, restart your daemons and check the settings.

# Downloads

* [Windows AMD64](https://github.com/TrueNodes/phantom/releases/latest/download/phantom-truenodes-windows-amd64.exe)
* [Linux AMD64](https://github.com/TrueNodes/phantom/releases/latest/download/phantom-truenodes-linux-amd64)
* [OSX AMD64](https://github.com/TrueNodes/phantom/releases/latest/download/phantom-truenodes-darwin-amd64)
* [ARMv7 Linux](https://github.com/TrueNodes/phantom/releases/latest/download/phantom-truenodes-linux-arm)
* [ARMv6 Pi](https://github.com/TrueNodes/phantom/releases/latest/download/phantom-truenodes-linux-armv6)
* [ARMv5 Pi](https://github.com/TrueNodes/phantom/releases/latest/download/phantom-truenodes-linux-armv5)

# Setup 

The setup is simple: copy your masternode.conf or fortunastake.conf, modify it slightly, launch the phantom executable.

## Masternode.txt setup

Copy your masternode.conf or fortunastake.conf to the same folder as the phantom executable. Rename it to masternode.txt. Remove any comment lines from the top of the file (i.e. delete any line starting with #). At the end of each line add a epoch time ( https://www.unixtimestamp.com ). The epoch timestamp is utilized to allow you to run multiple phantom node setups in a deterministic manner, creating a highly-available configuration.

**Example**

`masternode.conf`
```
# Masternode config file
# Format: alias IP:port masternodeprivkey collateral_output_txid collateral_output_index
mn1 45.50.22.125:17817 73HaYBVUCYjEMeeH1Y4sBGLALQZE1Yc1K64xiqgX37tGBDQL8Xg 2bcd3c84c84f87eaa86e4e56834c92927a07f9e18718810b92e0d0324456a67c 1
```

becomes:

`masternodes.txt`
```
mn1 45.50.22.125:17817 73HaYBVUCYjEMeeH1Y4sBGLALQZE1Yc1K64xiqgX37tGBDQL8Xg 2bcd3c84c84f87eaa86e4e56834c92927a07f9e18718810b92e0d0324456a67c 1 1555847365
```

comments removed, epoch timestamp added to the end.

## Run the phantom executable

```startphantom.sh```
```bash
while true; do
    ../bin/phantom-truenodes-linux-arm64 -coin_conf="coinconf.json" -masternode_conf="masternodes.txt" -db_path="peers.db" -max_connections=16 -min_connections=3 -broadcast_listen=true
sleep 30
done
```

## Coin configurations 
There is a coinconf generator included that can auto-generate settings for most masternode coins. Check the `tools/coinconf` directory or in releases

## Available Flags

```-bootstrap_hash``` string    
Hash to bootstrap the pings with ( top - 12 ) 

```-bootstrap_ips``` string     
IP address to bootstrap the network 

```-bootstrap_url``` string     
Explorer to bootstrap from. 

```-coin_conf``` string       
Name of the file to load the coin information from.   

```-daemon_version``` string    
The string to use for the sentinel version number (i.e. 1.20.0) (default "0.0.0.0") 

```-magic_message``` string 
The signing message 

```-magic_message_newline``` bool   
Add a new line to the magic message (default true)  

```-magicbytes``` string    
A hex string for the magic bytes    

```-masternode_conf``` string 
Name of the file to load the masternode information from. 

```-max_connections``` uint 
The number of peers to maintain (default 10)    

```-min_connections``` uint 
The minimum acceptable number of peers to maintain. If not satified in 5 minutes after app starts, then exit (default 0, never exit)  

```-noblock_minutes``` uint 
Maximum value, in minutes, without receiving block signaling from the network. If you don't receive it in that time, close the software. Start counting after 5 minutes software started. (default 0, never exit)   

```-port``` uint    
The default port number 

```-protocol_number``` uint 
The protocol number to connect and ping with    

```-sentinel_version``` string  
The string to use for the sentinel version number (i.e. 1.20.0) (default "0.0.0")   

```-user_agent``` string  
The user agent string to connect to remote peers with.    

```-db_path``` string 
The destination for peer database storage (default path is ./peers.db)    

## Building from source code

```
./build.sh 
```

## Donation Addresses (original dev, not TrueNodes
breakcrypto:    
BTC: 151HTde9NgwbMMbMmqqpJYruYRL4SLZg1S
