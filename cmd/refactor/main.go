package main

import (
	"flag"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"../../pkg/phantom"
	"../../pkg/socket/wire"
	"./analyzer"
	"./blockqueue"
	"./broadcaststore"
	"./coinconf"
	"./database"
	"./dnsseed"
	"./events"
	"./generator"
	"./remotechains"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
)

type PeerCollection struct {
	PeerConnections []*PeerConnection
}

type PhantomDaemon struct {
	MaxConnections  uint
	MinConnections  uint
	NoblockMinutes  uint
	BootstrapIPs    string
	DNSSeeds        string
	BootstrapHash   chainhash.Hash
	BootstrapChains []remotechains.RemoteChain
	MasternodeConf  string
	CoinCon         phantom.CoinConf
	DefaultPort     uint
	PeerConnections []database.Peer
	//for the peers
	PeerConnectionTemplate PeerConnection
}

func (p *PeerCollection) RemovePeer(peerToRemove *PeerConnection) {
	for i, peer := range p.PeerConnections {
		if peer.PeerInfo.Address == peerToRemove.PeerInfo.Address && peer.PeerInfo.Port == peerToRemove.PeerInfo.Port {
			p.PeerConnections = append(p.PeerConnections[:i], p.PeerConnections[i+1:]...)
		}
	}
}

func (p *PeerCollection) AddPeer(peerToAdd *PeerConnection) {
	p.PeerConnections = append(p.PeerConnections, peerToAdd)
}

func (p *PeerCollection) Contains(peerToCheck *database.Peer) bool {
	for _, peer := range p.PeerConnections {
		if peer.PeerInfo.Address == peerToCheck.Address && peer.PeerInfo.Port == peerToCheck.Port {
			return true
		}
	}
	return false
}

func (p *PeerCollection) CountPeers() int {
	var counter = 0
	for _, peer := range p.PeerConnections {
		if time.Now().Sub(peer.PeerInfo.LastSeen).Minutes() < 20 {
			counter++
		}
	}
	return counter
}

var peerCollection PeerCollection

func init() {
	// Only log the warning severity or above.
	//log.SetLevel(log.DebugLevel)
}

func main() {
	const VERSION = "2.0.0-beta"

	var done chan bool

	phantom.Preamble(VERSION)
	//time.Sleep(10 * time.Second)

	phantomDaemon := PhantomDaemon{}

	var magicHex string
	var magicMsgNewLine bool = true
	var protocolNum uint
	var bootstrapHashStr string
	var bootstrapChainsStr string
	var sentinelString string
	var daemonString string
	var coinConfString string
	var debugLogging bool

	flag.StringVar(&coinConfString, "coin_conf", "coinconf.json", "Name of the file to load the coin information from.")

	flag.StringVar(&phantomDaemon.MasternodeConf, "masternode_conf",
		"masternode.conf",
		"Name of the file to load the masternode information from.")

	flag.UintVar(&phantomDaemon.MaxConnections, "max_connections",
		10,
		"the number of peers to maintain")

	flag.UintVar(&phantomDaemon.MinConnections, "min_connections",
		0,
		"the minimum number of peers to maintain. 0 is disabled")

	flag.UintVar(&phantomDaemon.NoblockMinutes, "noblock_minutes",
		0,
		"the maximum number (in minutes) without receiving blocks. 0 is disabled")

	flag.StringVar(&magicHex, "magicbytes",
		"",
		"a hex string for the magic bytes")

	flag.UintVar(&phantomDaemon.DefaultPort, "port",
		0,
		"the default port number")

	flag.UintVar(&protocolNum, "protocol_number",
		0,
		"the protocol number to connect and ping with")

	flag.StringVar(&phantomDaemon.PeerConnectionTemplate.MagicMessage, "magic_message",
		"DarkNet Signed Message:",
		"the signing message")

	flag.BoolVar(&magicMsgNewLine,
		"magic_message_newline",
		true,
		"add a new line to the magic message")

	flag.StringVar(&phantomDaemon.BootstrapIPs, "bootstrap_ips",
		"",
		"IP addresses to bootstrap the network (i.e. \"1.1.1.1:1234,2.2.2.2:1234\")")

	flag.StringVar(&phantomDaemon.DNSSeeds, "dns_seeds",
		"",
		"DNS seed addresses to bootstrap the network (i.e. \"dns.coin.com,dns1.coin.net\")")

	flag.StringVar(&bootstrapHashStr, "bootstrap_hash",
		"",
		"Hash to bootstrap the pings with ( top - 12 )")

	flag.StringVar(&bootstrapChainsStr,
		"bootstrap_chains",
		"",
		`Remote chains to bootstrap from. This is a JSON array in the form of:]\n)
			[{"username:"user","password":"secret","format":"iquidus","url":"http://some.explorer"},{...}"]\n\n
			Valid chain formats are: iquidus, insight, or rpc`)

	flag.StringVar(&sentinelString,
		"sentinel_version",
		"",
		"The string to use for the sentinel version number (i.e. 1.20.0)")

	flag.StringVar(&daemonString,
		"daemon_version",
		"",
		"The string to use for the sentinel version number (i.e. 1.20.0)")

	flag.StringVar(&phantomDaemon.PeerConnectionTemplate.UserAgent,
		"user_agent",
		"TrueNodes - Masternode Hosting",
		"The user agent string to connect to remote peers with.")

	flag.BoolVar(&phantomDaemon.PeerConnectionTemplate.BroadcastListen,
		"broadcast_listen",
		true,
		"If set to true, the phantom will listen for new broadcasts and cache them for 4 hours.")

	flag.BoolVar(&phantomDaemon.PeerConnectionTemplate.Autosense,
		"autosense",
		true,
		"If set to true, the phantom will listen for new broadcasts and cache them for 4 hours.")

	flag.BoolVar(&debugLogging,
		"debug",
		false,
		"Enable debug output.")

	flag.Parse()

	if debugLogging {
		log.SetLevel(log.DebugLevel)
	}

	var coinConf = coinconf.CoinConf{}

	if coinConfString != "" {
		var err error
		coinConf, err = coinconf.LoadCoinConf(coinConfString)
		if err != nil {
			log.Fatal(err)
		} else {
			if phantomDaemon.MasternodeConf == "" {
				phantomDaemon.MasternodeConf = coinConf.MasternodeConf
			}

			if coinConf.MaxConnections != nil {
				phantomDaemon.MaxConnections = uint(*coinConf.MaxConnections)
			}

			if coinConf.MinConnections != nil {
				phantomDaemon.MinConnections = uint(*coinConf.MinConnections)
			}

			if coinConf.NoblockMinutes != nil {
				phantomDaemon.NoblockMinutes = uint(*coinConf.NoblockMinutes)
			}

			if magicHex == "" {
				magicHex = coinConf.Magicbytes
			}

			if phantomDaemon.DefaultPort == 0 {
				phantomDaemon.DefaultPort = uint(coinConf.Port)
			}

			if protocolNum == 0 {
				protocolNum = uint(coinConf.ProtocolNumber)
			}

			if phantomDaemon.PeerConnectionTemplate.MagicMessage == "" || coinConf.MagicMessage != "" {
				phantomDaemon.PeerConnectionTemplate.MagicMessage = coinConf.MagicMessage
			}

			if magicMsgNewLine && coinConf.MagicMessageNewline != nil {
				magicMsgNewLine = *coinConf.MagicMessageNewline
			}

			if phantomDaemon.BootstrapIPs == "" {
				phantomDaemon.BootstrapIPs = coinConf.BootstrapIPs
			}

			if phantomDaemon.DNSSeeds == "" {
				phantomDaemon.DNSSeeds = coinConf.DNSSeeds
			}

			if bootstrapHashStr == "" || coinConf.BootstrapHash != "" {
				bootstrapHashStr = coinConf.BootstrapHash
			}

			if bootstrapChainsStr == "" {
				bootstrapChainsStr = coinConf.BootstrapChains
			}

			if sentinelString == "" {
				sentinelString = coinConf.SentinelVersion
			}

			if daemonString == "" {
				daemonString = coinConf.DaemonVersion
			}

			if phantomDaemon.PeerConnectionTemplate.UserAgent == "TrueNodes - Masternode Hosting" ||
				coinConf.UserAgent != "" {
				phantomDaemon.PeerConnectionTemplate.UserAgent = coinConf.UserAgent
			}

			if !phantomDaemon.PeerConnectionTemplate.BroadcastListen || coinConf.BroadcastListen != nil {
				phantomDaemon.PeerConnectionTemplate.BroadcastListen = *coinConf.BroadcastListen
			}

			if phantomDaemon.PeerConnectionTemplate.Autosense || coinConf.Autosense != nil {
				phantomDaemon.PeerConnectionTemplate.Autosense = *coinConf.Autosense
			}
		}
	}

	magicBytes64, _ := strconv.ParseUint(magicHex, 16, 32)
	phantomDaemon.PeerConnectionTemplate.MagicBytes = uint32(magicBytes64)

	phantomDaemon.PeerConnectionTemplate.ProtocolNumber = uint32(protocolNum)

	if sentinelString != "" {
		phantomDaemon.PeerConnectionTemplate.SentinelVersion = phantom.ConvertVersionStringToInt(sentinelString)
	}

	if daemonString != "" {
		//fmt.Println("ENABLING DAEMON.")
		phantomDaemon.PeerConnectionTemplate.DaemonVersion = phantom.ConvertVersionStringToInt(daemonString)
	}

	if magicMsgNewLine {
		phantomDaemon.PeerConnectionTemplate.MagicMessage = phantomDaemon.PeerConnectionTemplate.MagicMessage + "\n"
	}

	if phantomDaemon.BootstrapIPs != "" {
		database.GetInstance().StorePeers(phantom.SplitAddressList(phantomDaemon.BootstrapIPs))
	}

	if bootstrapHashStr != "" {
		chainhash.Decode(&phantomDaemon.BootstrapHash, bootstrapHashStr)
	}

	if bootstrapChainsStr != "" {
		remoteChains, err := remotechains.ParseRemoteChains(bootstrapChainsStr)
		if err != nil {
			log.Warn("Failed to parse bootstrap_chains: ", bootstrapChainsStr)
		}

		for _, remote := range remoteChains {
			phantomDaemon.BootstrapChains = append(phantomDaemon.BootstrapChains,
				remotechains.StringToRemoteChain(remote.Format,
					remote.URL,
					remote.Username,
					remote.Password))
		}
	}

	log.WithFields(log.Fields{
		"masternode_conf": phantomDaemon.MasternodeConf,
		"min_connections": phantomDaemon.MinConnections,
		"max_connections": phantomDaemon.MaxConnections,
		"noblock_minutes": phantomDaemon.NoblockMinutes,
		"magic_bytes": strings.ToUpper(strconv.FormatInt(
			int64(phantomDaemon.PeerConnectionTemplate.MagicBytes), 16)),
		"magic_message":    phantomDaemon.PeerConnectionTemplate.MagicMessage,
		"protocol_number":  phantomDaemon.PeerConnectionTemplate.ProtocolNumber,
		"bootstrap_ips":    phantomDaemon.BootstrapIPs,
		"bootstrap_chains": bootstrapChainsStr,
		"bootstrap_hash":   phantomDaemon.BootstrapHash.String(),
		"autosense":        phantomDaemon.PeerConnectionTemplate.Autosense,
		"broadcast_listen": phantomDaemon.PeerConnectionTemplate.BroadcastListen,
		"daemon_version":   phantomDaemon.PeerConnectionTemplate.DaemonVersion,
		"sentinel_version": phantomDaemon.PeerConnectionTemplate.SentinelVersion,
		"user_agent":       phantomDaemon.PeerConnectionTemplate.UserAgent,
		"dns_seeds":        phantomDaemon.DNSSeeds,
		"default_port":     phantomDaemon.DefaultPort,
		"debug":            debugLogging,
	}).Info("Using the following settings.")

	phantomDaemon.Start()

	//wait
	<-done
}

func (p *PhantomDaemon) Start() {

	//load the peer database
	peerdb := database.GetInstance()
	queue := blockqueue.GetInstance()

	//allocate the peer channels
	peerChannels := p.allocatePeerChannels(p.MaxConnections)

	//start monitoring for cases that require restart (noblock_minutes and min_connections)
	go p.MonitorForNeededRestart()

	//setup the event channel that all peers will broadcast to
	var daemonEventChannel = make(chan events.Event)

	//load the bootstrap values
	//load blockhash
	//load peers

	var hash chainhash.Hash
	defaultHash := chainhash.Hash{}

	for _, bootstrap := range p.BootstrapChains {
		peers, err := bootstrap.GetPeers(uint32(p.DefaultPort))
		if err != nil {
			log.Error("Failed to load bootstrap peers")
		}
		peerdb.StorePeers(peers)

		height, err := bootstrap.GetChainHeight()
		if err != nil {
			log.Error("Failed to load bootstrap height")
		}

		//make sure we haven't loaded a bootstrap already
		if hash == defaultHash {
			hash, err = bootstrap.GetBlockHash(height - 12)
			if err != nil {
				log.Error("Failed to load bootstrap hash")
				hash = defaultHash //reset to default just to be safe
			}
			log.WithFields(log.Fields{
				"hash": hash.String(),
			}).Info("Bootstrap hash value")
		}
	}

	if hash == defaultHash {
		hash = p.BootstrapHash
	}

	//force the bootstrap
	queue.ForceHash(hash)

	//load the dnsseeds if there are any
	if p.DNSSeeds != "" {
		for _, seed := range strings.Split(p.DNSSeeds, ",") {
			peerdb.StorePeers(dnsseed.LoadDnsSeeds(seed, uint32(p.DefaultPort)))
		}
	}

	//start the analyzer
	mnpAnalyzer := analyzer.GetInstance()
	mnpAnalyzer.Threshold = 10

	//start processing events before spawning off peers
	go p.processEvents(daemonEventChannel)

	//spawn off the peers
	peers := peerdb.GetRandomPeers(p.MaxConnections)
	for i, peer := range peers {
		peerConn := PeerConnection{
			MagicBytes:        p.PeerConnectionTemplate.MagicBytes,
			ProtocolNumber:    p.PeerConnectionTemplate.ProtocolNumber,
			MagicMessage:      p.PeerConnectionTemplate.MagicMessage,
			PeerInfo:          peer,
			InboundEvents:     peerChannels[i],
			OutboundEvents:    daemonEventChannel,
			SentinelVersion:   p.PeerConnectionTemplate.SentinelVersion,
			DaemonVersion:     p.PeerConnectionTemplate.DaemonVersion,
			UseOutpointFormat: p.PeerConnectionTemplate.UseOutpointFormat,
			Autosense:         p.PeerConnectionTemplate.Autosense,
			BroadcastListen:   p.PeerConnectionTemplate.BroadcastListen,
		}

		log.WithField("peer ip", peer).Debug("Starting new peer.")
		go peerConn.Start(&hash, "break")
		peerCollection.AddPeer(&peerConn)
	}

	//start the ping generator
	//auto-sense enabled?
	//if so don't start sending pings until we've reached consensus

	//TODO Make this a channel gate vs. polling
	for p.PeerConnectionTemplate.Autosense {
		//do nothing until the auto-sense finishes
		time.Sleep(time.Second * 10)
		log.Info("Sleeping, waiting on autosense to complete.")
	}

	time.Sleep(time.Second * 30)

	go p.generatePings(peerChannels...)
}

func (p *PhantomDaemon) processEvents(eventChannel chan events.Event) {
	for event := range eventChannel {
		//process the event
		switch event.Type {
		case events.NewMasternodePing:
			mnp := event.Data.(*wire.MsgMNP)
			p.processNewMasternodePing(mnp)
		case events.NewMasternodeBroadcast:
			mnb := event.Data.(*wire.MsgMNB)
			broadcaststore.GetInstance().StoreBroadcast(mnb)
		case events.NewBlock:
			hash := event.Data.(*chainhash.Hash)
			//log.WithField("hash", hash.String()).Info("New block.")
			blockqueue.GetInstance().AddHash(*hash)
		case events.NewAddr:
			addr := event.Data.(*wire.NetAddress)
			//log.WithField("addr", addr.IP).Debug("New address found. Saving.")
			database.GetInstance().StorePeer(database.Peer{Address: addr.IP.String(), Port: uint32(addr.Port), LastSeen: time.Now()})
		case events.PeerDisconnect:
			peer := event.Data.(*PeerConnection)
			log.WithField("ip", peer.PeerInfo.Address).Debug("Handled peer disconnection.")
			p.processPeerDisconenct(peer)
		}
	}
}

func (p *PhantomDaemon) allocatePeerChannels(numConn uint) []chan events.Event {
	var peerChannels []chan events.Event
	for i := 0; i < int(numConn); i++ {
		peerChannels = append(peerChannels, make(chan events.Event, 1500))
	}
	return peerChannels
}

func (p *PhantomDaemon) processNewMasternodePing(ping *wire.MsgMNP) {
	log.Debug("Analyzing ping.")
	//we have a new ping
	if analyzer.GetInstance().AnalyzePing(ping) {
		//we have enough to analyze and make a decent guess at the network settings
		useOut, sentinel, daemon := analyzer.GetInstance().GetResults()

		log.Info("-------------------------")
		log.Info("--- CONSENSUS REACHED ---")
		log.Info("-------------------------")
		log.WithFields(log.Fields{
			"Outpoint form":    useOut,
			"Sentinel version": sentinel,
			"Daemon version":   daemon,
		}).Info("Consensus reached.")
		log.Info("-------------------------")
		log.Info("-------------------------")

		//set the daemon settings for new peer connections
		p.PeerConnectionTemplate.Autosense = false
		p.PeerConnectionTemplate.UseOutpointFormat = useOut
		p.PeerConnectionTemplate.SentinelVersion = sentinel
		p.PeerConnectionTemplate.DaemonVersion = daemon

		//update existing peers
		for _, peer := range peerCollection.PeerConnections {
			peer.Autosense = false
			peer.UseOutpointFormat = p.PeerConnectionTemplate.UseOutpointFormat
			peer.SentinelVersion = p.PeerConnectionTemplate.SentinelVersion
			peer.DaemonVersion = p.PeerConnectionTemplate.DaemonVersion
		}
	}
}

func (p *PhantomDaemon) generatePings(channels ...chan events.Event) {
	for {
		startTime := time.Now()

		generator.GeneratePingsFromMasternodeFile(
			p.MasternodeConf,
			p.PeerConnectionTemplate.MagicMessage,
			p.PeerConnectionTemplate.UseOutpointFormat,
			p.PeerConnectionTemplate.SentinelVersion,
			p.PeerConnectionTemplate.DaemonVersion,
			channels...,
		)

		sleepTime := startTime.Add(time.Minute * 10).Sub(time.Now())

		//sleep for the remaining 10 minutes, if there are any.
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}
}

func (p *PhantomDaemon) processPeerDisconenct(peer *PeerConnection) {
	//a peer has closed out
	peerCollection.RemovePeer(peer)

	//turn the channel to not accepting new events (wrap, set a status code)

	//bleed the events out of the channel if they exist

	//now take the pinger channel from it and reuse in a newly created peer
	newPeer := p.spawnNewPeer(peer.InboundEvents, peer.OutboundEvents)
	peerCollection.AddPeer(newPeer)
	go newPeer.Start(blockqueue.GetInstance().GetTop(), "break")
}

func (p *PhantomDaemon) spawnNewPeer(inboundEventChannel chan events.Event, outboundEventChannel chan events.Event) *PeerConnection {
	//get a new address from the database
	peerdb := database.GetInstance()

	var peerInfo database.Peer
	for peerInfo = peerdb.GetRandomPeer(); peerCollection.Contains(&peerInfo); peerInfo = peerdb.GetRandomPeer() {
		//loop until it fills
		//fmt.Println(peerInfo.Address)
	}

	log.WithField("ip", peerInfo.Address).Debug("Spawned new peer.")

	//drain the channel before assigning to remove any stale pings
	drainChannel(inboundEventChannel)

	peer := PeerConnection{
		MagicBytes:        p.PeerConnectionTemplate.MagicBytes,
		ProtocolNumber:    p.PeerConnectionTemplate.ProtocolNumber,
		MagicMessage:      p.PeerConnectionTemplate.MagicMessage,
		PeerInfo:          peerInfo,
		InboundEvents:     inboundEventChannel,
		OutboundEvents:    outboundEventChannel,
		SentinelVersion:   p.PeerConnectionTemplate.SentinelVersion,
		DaemonVersion:     p.PeerConnectionTemplate.DaemonVersion,
		UseOutpointFormat: p.PeerConnectionTemplate.UseOutpointFormat,
		Autosense:         p.PeerConnectionTemplate.Autosense,
		BroadcastListen:   p.PeerConnectionTemplate.BroadcastListen,
	}

	return &peer
}

func drainChannel(channel chan events.Event) {
	for len(channel) > 0 {
		<-channel
	}
}

var StartTime time.Time = time.Now()
var LastBlockTime time.Time = time.Now().Add(time.Minute * 5)
var LastPingtime time.Time = time.Now().Add(time.Minute * 5)

func (p *PhantomDaemon) MonitorForNeededRestart() {
	StartTime = time.Now()

	for {
		//check every minute
		time.Sleep(time.Minute * 1)
		var num = peerCollection.CountPeers()
	
		//active is LastSeen in minus than 20 minutes ago
		log.Info("Active Connections: ", num, " (total: ", len(peerCollection.PeerConnections),", min: ", p.MinConnections, ", max: ", p.MaxConnections, ")")
		if p.MinConnections > 0 && num < int(p.MinConnections) && time.Now().Sub(StartTime).Minutes() > 5 {
			runningTime := time.Now().Sub(StartTime)
			withoutPingTime := time.Now().Sub(LastPingtime)
			log.Error("Under minimum number of connections (even 5 minutes after phantom start). Phantom running for ", math.Floor(runningTime.Hours()/24), "d ", math.Floor(math.Remainder(runningTime.Hours(), 24)), "h ", math.Floor(math.Remainder(runningTime.Minutes(), 24)), "m ", math.Floor(math.Remainder(runningTime.Seconds(), 24)), "s ", "Without ping for ", math.Floor(withoutPingTime.Hours()/24), "d ", math.Floor(math.Remainder(withoutPingTime.Hours(), 24)), "h ", math.Floor(math.Remainder(withoutPingTime.Minutes(), 24)), "m ", math.Floor(math.Remainder(withoutPingTime.Seconds(), 24)), "s. CLOSING APPLICATION NOW")
			os.Exit(0)
		}
		
		if p.NoblockMinutes > 0 && math.Floor(math.Abs(time.Now().Sub(LastBlockTime).Minutes())) > float64(p.NoblockMinutes) {
			runningTime := time.Now().Sub(StartTime)
			withoutNewBlocks := time.Now().Sub(LastBlockTime)
			log.Error("More than ", p.NoblockMinutes, " minutes without receiving new blocks. Phantom running for ", math.Floor(runningTime.Hours()/24), "d ", math.Floor(math.Remainder(runningTime.Hours(), 24)), "h ", math.Floor(math.Remainder(runningTime.Minutes(), 24)), "m ", math.Floor(math.Remainder(runningTime.Seconds(), 24)), "s ", "Without new blocks for ", math.Floor(withoutNewBlocks.Hours()/24), "d ", math.Floor(math.Remainder(withoutNewBlocks.Hours(), 24)), "h ", math.Floor(math.Remainder(withoutNewBlocks.Minutes(), 24)), "m ", math.Floor(math.Remainder(withoutNewBlocks.Seconds(), 24)), "s. CLOSING APPLICATION NOW")
			os.Exit(0)
		}
	}
}
