/**
*    Copyright (C) 2019-present C2CV Holdings, LLC.
*
*    This program is free software: you can redistribute it and/or modify
*    it under the terms of the Server Side Public License, version 1,
*    as published by C2CV Holdings, LLC.
*
*    This program is distributed in the hope that it will be useful,
*    but WITHOUT ANY WARRANTY; without even the implied warranty of
*    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*    Server Side Public License for more details.
*
*    You should have received a copy of the Server Side Public License
*    along with this program. If not, see
*    <http://www.mongodb.com/licensing/server-side-public-license>.
*
*    As a special exception, the copyright holders give permission to link the
*    code of portions of this program with the OpenSSL library under certain
*    conditions as described in each individual source file and distribute
*    linked combinations including the program with the OpenSSL library. You
*    must comply with the Server Side Public License in all respects for
*    all of the code used other than as permitted herein. If you modify file(s)
*    with this exception, you may extend this exception to your version of the
*    file(s), but you are not obligated to do so. If you do not wish to do so,
*    delete this exception statement from your version. If you delete this
*    exception statement from all source files in the program, then also delete
*    it in the license file.
 */

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"../../pkg/phantom"
	"../../pkg/socket/wire"
	"../../pkg/storage"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

var minConnections uint
var maxConnections uint
var noBlockMinutes uint

var magicBytes uint32
var defaultPort uint
var protocolNumber uint32
var magicMessage string
var bootstrapIPs string
var bootstrapHash chainhash.Hash
var bootstrapExplorer string
var sentinelVersion uint32
var daemonVersion uint32
var masternodeConf string
var coinCon phantom.CoinConf
var dbPath string
var userAgent string
var cachedPeers error
var numberConnections int

const VERSION = "1.2.10"

var StartTime time.Time

func main() {

	//disable all logging
	//log.SetOutput(ioutil.Discard)

	StartTime := time.Now()

	var magicHex string
	var magicMsgNewLine bool
	var protocolNum uint
	var bootstrapHashStr string
	var sentinelString string
	var daemonString string
	var coinConfString string
	var broadcastListen bool

	flag.StringVar(&coinConfString, "coin_conf", "coinconf.json", "Name of the file to load the coin information from.")
	flag.StringVar(&masternodeConf, "masternode_conf", "masternodeconf.json", "Name of the file to load the masternode information from.")
	flag.UintVar(&minConnections, "min_connections", 0, "the minimum acceptable number of peers to maintain. If not satified in 5 minutes after app starts, then exit (default 0, never exit)")
	flag.UintVar(&maxConnections, "max_connections", 64, "the maximum number of peers to maintain")
	flag.UintVar(&noBlockMinutes, "noblock_minutes", 0, "Maximum value, in minutes, without receiving block signaling from the network. If you don't receive it in that time, close the software. Start counting after 5 minutes software started. (default 0, never exit)")
	flag.StringVar(&magicHex, "magicbytes", "", "a hex string for the magic bytes")
	flag.UintVar(&defaultPort, "port", 0, "the default port number")
	flag.UintVar(&protocolNum, "protocol_number", 0, "the protocol number to connect and ping with")
	flag.StringVar(&magicMessage, "magic_message", "", "the signing message")
	flag.BoolVar(&magicMsgNewLine, "magic_message_newline", true, "add a new line to the magic message")
	flag.StringVar(&bootstrapIPs, "bootstrap_ips", "", "IP addresses to bootstrap the network (i.e. \"1.1.1.1:1234,2.2.2.2:1234\")")
	flag.StringVar(&bootstrapHashStr, "bootstrap_hash", "", "Hash to bootstrap the pings with ( top - 12 )")
	flag.StringVar(&bootstrapExplorer, "bootstrap_url", "", "Explorer to bootstrap from.")
	flag.StringVar(&sentinelString, "sentinel_version", "", "The string to use for the sentinel version number (i.e. 1.20.0)")
	flag.StringVar(&daemonString, "daemon_version", "", "The string to use for the sentinel version number (i.e. 1.20.0)")
	flag.StringVar(&userAgent, "user_agent", "True Nodes - Hospedagem de Masternodes", "The user agent string to connect to remote peers with.")
	flag.BoolVar(&broadcastListen, "broadcast_listen", true, "If set to true, the phantom will listen for new broadcasts and cache them for 4 hours.")
	flag.StringVar(&dbPath, "db_path", "./peers.db", "The destination for database storage.")
	flag.Parse()

	if coinConfString != "" {
		coinInfo, err := phantom.LoadCoinConf(coinConfString)
		if err != nil {
			log.Println("Error reading coin configuration information from:", coinConfString)
		} else {
			//load all the flags with the coin conf information
			//only overwrite default values
			if magicHex == "" {
				magicHex = coinInfo.Magicbytes
			}
			if defaultPort == 0 {
				defaultPort = coinInfo.Port
			}
			if protocolNum == 0 {
				protocolNum = coinInfo.ProtocolNumber
			}
			if magicMessage == "" {
				magicMessage = coinInfo.MagicMessage
			}
			if magicMsgNewLine && !coinInfo.MagicMessageNewline {
				magicMsgNewLine = false
			}
			if bootstrapIPs == "" {
				bootstrapIPs = coinInfo.BootstrapIPs
			}
			if bootstrapExplorer == "" {
				bootstrapExplorer = coinInfo.BootstrapURL
			}
			if sentinelString == "" {
				sentinelString = coinInfo.SentinelVersion
			}
			if daemonString == "" {
				daemonString = coinInfo.DaemonVersion
			}
			if userAgent == "True Nodes - Hospedagem de Masternodes" && coinInfo.UserAgent != "" {
				userAgent = coinInfo.UserAgent
			}
			if dbPath == "" {
				dbPath = "peers.db"
			}
		}
	}

	magicMsgNewLine = true

	magicBytes64, _ := strconv.ParseUint(magicHex, 16, 32)
	magicBytes = uint32(magicBytes64)

	protocolNumber = uint32(protocolNum)

	if sentinelString != "" {
		//fmt.Println("ENABLING SENTINEL.")
		sentinelVersion = phantom.ConvertVersionStringToInt(sentinelString)
	}

	if daemonString != "" {
		//fmt.Println("ENABLING DAEMON.")
		daemonVersion = phantom.ConvertVersionStringToInt(daemonString)
	}

	if magicMsgNewLine {
		magicMessage = magicMessage + "\n"
	}

	var connectionSet = make(map[string]*phantom.PingerConnection)
	var peerSet = make(map[string]wire.NetAddress)
	var broadcastSet = make(map[string]wire.MsgMNB)

	var waitGroup sync.WaitGroup

	if bootstrapIPs != "" {
		addresses := phantom.SplitAddressList(bootstrapIPs)

		if uint(len(addresses)) > maxConnections {
			addresses = addresses[:maxConnections-1]
		}

		for _, address := range addresses {
			peerSet[address.IP.String()] = address
		}
	}

	addrProcessingChannel := make(chan wire.NetAddress, 1500)
	hashProcessingChannel := make(chan chainhash.Hash, 1500)

	var broadcastProcessingChannel chan wire.MsgMNB
	if broadcastListen {
		broadcastProcessingChannel = make(chan wire.MsgMNB, 1500)
	}

	hashQueue := phantom.NewQueue(12)

	if bootstrapExplorer != "" {
		//check for a trailing slash
		if bootstrapExplorer[len(bootstrapExplorer)-1] == '/' {
			bootstrapExplorer = bootstrapExplorer[0 : len(bootstrapExplorer)-1]
		}

		bootstrapper := phantom.Bootstrapper{bootstrapExplorer}
		var err error
		bootstrapHash, err = bootstrapper.LoadBlockHash()

		empthHash := chainhash.Hash{}
		if err != nil {
			log.Fatal("Unable to bootstrap using the explorer url provided. ", err)
		}

		if bootstrapHash == empthHash {
			log.Fatal("Unable to bootstrap using the explorer url provided. Invalid result returned.")
		}

		peers, _ := bootstrapper.LoadPossiblePeers(uint16(defaultPort))

		for _, peer := range peers {
			if len(peerSet) < int(maxConnections) {
				peerSet[peer.IP.String()] = peer
			} else {
				break //exit early
			}
		}

	} else {
		chainhash.Decode(&bootstrapHash, bootstrapHashStr)
		hashQueue.Push(&bootstrapHash)
	}

	phantom.Preamble(VERSION)

	time.Sleep(10 * time.Second)

	fmt.Println("--USING THE FOLLOWING SETTINGS--")
	fmt.Println("Coin configuration: ", coinConfString)
	fmt.Println("Masternode configuration: ", masternodeConf)
	fmt.Println("Magic Bytes: ", magicHex)
	fmt.Println("Magic Message: ", magicMessage)
	fmt.Println("Magic Message Newline: ", magicMsgNewLine)
	fmt.Println("Protocol Number: ", protocolNumber)
	fmt.Println("Bootstrap IPs: ", bootstrapIPs)
	fmt.Println("Default Port: ", defaultPort)
	fmt.Println("Hash: ", bootstrapHash)
	fmt.Println("Sentinel Version: ", sentinelVersion)
	fmt.Println("Daemon Version: ", daemonVersion)
	fmt.Println("Listen for broadcasts: ", broadcastListen)
	fmt.Println()
	fmt.Println("Minimum connections: ", minConnections)
	fmt.Println("Maximum connections: ", maxConnections)
	fmt.Println("Maximum time without blocks: ", noBlockMinutes, " minutes")
	fmt.Println("\n\n")

	db, err := storage.InitialiseDB(dbPath)
	if err != nil {
		log.Fatal("An error occurred initialising the database")
	} else {
		log.Println("Database was initialised:", db)
	}

	cachedPeers = storage.LoadPeersFromDB(db)

	for _, ip := range peerSet {
		//make the ping channel
		pingChannel := make(chan phantom.MasternodePing, 1500)

		waitGroup.Add(1)

		pinger := phantom.PingerConnection{
			MagicBytes:      magicBytes,
			IpAddress:       ip.IP.String(),
			Port:            uint16(ip.Port),
			ProtocolNumber:  protocolNumber,
			SentinelVersion: sentinelVersion,
			DaemonVersion:   daemonVersion,
			BootstrapHash:   bootstrapHash,
			PingChannel:     pingChannel,
			AddrChannel:     addrProcessingChannel,
			HashChannel:     hashProcessingChannel,
			Status:          0,
			WaitGroup:       &waitGroup,
		}

		//not ideal
		if broadcastListen {
			pinger.BroadcastChannel = broadcastProcessingChannel
		}

		//make a client
		connectionSet[pinger.IpAddress] = &pinger

		go pinger.Start(userAgent)
	}

	pingGeneratorChannel := make(chan phantom.MasternodePing, 1500)

	waitGroup.Add(1)

	if broadcastListen {
		go processNewBroadcasts(broadcastProcessingChannel, broadcastSet)
	}

	go processNewAddresses(addrProcessingChannel, peerSet)
	go processNewHashes(hashProcessingChannel, hashQueue)
	go sendPings(connectionSet, peerSet, pingGeneratorChannel, addrProcessingChannel, hashProcessingChannel, broadcastProcessingChannel, waitGroup)
	go generatePings(pingGeneratorChannel, hashQueue, magicMessage, broadcastSet)

	waitGroup.Wait()

	elapsed := time.Since(StartTime)
	fmt.Println("Started in %s", elapsed)

	StartTime = time.Now()
}

func generatePings(pingChannel chan phantom.MasternodePing, queue *phantom.Queue,
	magicMessage string, broadcastSet map[string]wire.MsgMNB) {

	for {
		phantom.GeneratePingsFromMasternodeFile(
			masternodeConf,
			pingChannel,
			queue,
			magicMessage,
			sentinelVersion,
			daemonVersion,
			broadcastSet,
		)

		time.Sleep((time.Minute * 10) + (time.Second * 5))
	}
}

func processNewHashes(hashChannel chan chainhash.Hash, queue *phantom.Queue) {
	for {
		hash := <-hashChannel

		//log.Println("Adding hash to queue: ", hash.String(), "(", queue.count, ")")

		queue.Push(&hash)
		for queue.Len() > 12 { //clear the queue until we're at 12 entries
			queue.Pop()
			//log.Println("Removing hash from queue: ", popped.String(), "(", queue.count, ")")
		}
	}
}

func processNewBroadcasts(broadcastChannel chan wire.MsgMNB, broadcastSet map[string]wire.MsgMNB) {
	for {

		mnb := <-broadcastChannel

		broadcastSet[mnb.Vin.PreviousOutPoint.Hash.String()+
			":"+strconv.Itoa(int(mnb.Vin.PreviousOutPoint.Index))] = mnb
	}
}

func processNewAddresses(addrChannel chan wire.NetAddress, peerSet map[string]wire.NetAddress) {
	// Cache peers
	db, err := storage.InitialiseDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	for {
		addr := <-addrChannel

		if addr.IP.To4() == nil {
			continue
		}

		peerSet[addr.IP.String()] = addr
		err = storage.CachePeerToDB(db, addr.IP.String())
	}
}

func getNextPeer(connectionSet map[string]*phantom.PingerConnection, peerSet map[string]wire.NetAddress) (returnValue wire.NetAddress, err error) {
	for peer := range peerSet {
		if _, ok := connectionSet[peer]; !ok {
			//we have a peer that isn't in the conncetion list return it
			returnValue = peerSet[peer]

			//remove the peer from the connection list
			delete(peerSet, peer)
			log.Println("New peer found: ", peer, " (", numberConnections, "/", maxConnections, ")")

			return returnValue, nil
		}
	}
	return returnValue, errors.New("No peers found.")
}

func sendPings(connectionSet map[string]*phantom.PingerConnection, peerSet map[string]wire.NetAddress, pingChannel chan phantom.MasternodePing, addrChannel chan wire.NetAddress, hashChannel chan chainhash.Hash, broadcastChannel chan wire.MsgMNB, waitGroup sync.WaitGroup) {

	time.Sleep(10 * time.Second) //hack to work around .Wait() race condition on fast start-ups

	for {
		ping := <-pingChannel

		sleepTime := ping.PingTime.Sub(time.Now())

		t := ping.PingTime.UTC()

		if sleepTime > 0 {
			// log.Println(ping.Name, t.Format("15:04:05"), "sleeping for ", sleepTime.String())
			time.Sleep(sleepTime)
		} else {
			log.Println(ping.Name, t.Format("15:04:05"), "awake")
		}

		//send the ping
		// Iterate through list and print its contents.
		var newConnectionSet = make(map[string]*phantom.PingerConnection)

		for _, pinger := range connectionSet {
			status := pinger.GetStatus()

			if status < 0 || len(pinger.PingChannel) > 10 { //the pinger has had an error, close the channel
				fmt.Println("There's been an error, closing connection to ", pinger.IpAddress)
				pinger.SetStatus(-1)

				log.Printf("%s : Closing down the ping channel.\n", pinger.IpAddress)
				close(pinger.PingChannel) // don't add the closed pinger to the connectionArray

				//remove the peer from the peerSet
				delete(peerSet, pinger.IpAddress)
			} else {
				if status > 0 {
					log.Printf("%s : Pinging.", pinger.IpAddress)
					pinger.PingChannel <- ping //only ping on connected pingers (1)
				}
				// this filters out bad connections, re-add unconnected peers just to be safe
				// log.Printf("Re-added %s to the queue (channel #: %d).\n", pinger.IpAddress, len(pinger.PingChannel))
				newConnectionSet[pinger.IpAddress] = pinger
			}
		}

		//replace the pointer
		connectionSet = newConnectionSet
		numberConnections = len(connectionSet)

		//spawn off extra nodes here if we don't have enough
		if len(connectionSet) < int(maxConnections) {

			//	log.Println("Under the max connection count, spawning new peer (", len(connectionSet), " / ", maxConnections, ")")

			for i := 0; i < int(maxConnections)-len(connectionSet); i++ {

				//spawn off a new connection
				peer, err := getNextPeer(connectionSet, peerSet)

				if err != nil {
					//	log.Println("No new peers found.")
					continue
				}

				newPingChannel := make(chan phantom.MasternodePing, 1500)

				// intentionally don't provide a bootstraphash to prevent
				// duplicate data downloads for unneeded blocks
				newPinger := phantom.PingerConnection{
					MagicBytes:      magicBytes,
					IpAddress:       peer.IP.String(),
					Port:            peer.Port,
					ProtocolNumber:  protocolNumber,
					SentinelVersion: sentinelVersion,
					DaemonVersion:   daemonVersion,
					PingChannel:     newPingChannel,
					AddrChannel:     addrChannel,
					HashChannel:     hashChannel,
					Status:          0,
					WaitGroup:       &waitGroup,
				}

				if broadcastChannel != nil {
					newPinger.BroadcastChannel = broadcastChannel
				}

				//make a client
				newConnectionSet[newPinger.IpAddress] = &newPinger
				//connectionList = nil //release for the GC

				waitGroup.Add(1)
				go newPinger.Start(userAgent)

				fmt.Println("Opened a new connection to ", newPinger.IpAddress, " (", numberConnections, "/", maxConnections, ")")
			}
		}

		if numberConnections > 0 && numberConnections < int(minConnections) && time.Now().Sub(StartTime).Seconds() > 300 {
			var runningTime time.Duration = time.Now().Sub(StartTime)

			log.Println("Minimum number of connections (", minConnections, ") not satisfied. Application has been running for ", runningTime)
			log.Println("Closing Application now")

			os.Exit(0)
		}

		if noBlockMinutes > 0 && time.Now().Sub(phantom.LastBlockTime).Minutes() > float64(noBlockMinutes) {
			var runningTime time.Duration = time.Now().Sub(StartTime)

			log.Println("More than ", noBlockMinutes, " minutes without receiving blocks from network. Application has been running for ", runningTime)
			log.Println("Closing Application now")

			os.Exit(0)
		}
	}

	waitGroup.Done()
}
