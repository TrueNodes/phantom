package remotechains

import (
	"encoding/json"
	"../../../pkg/socket/wire"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"log"
	"net"
	"../database"
	"strconv"
	"strings"
	"time"
)

const (
	UNKNOWN RemoteChainFormat = -1
	IQUIDUS RemoteChainFormat = 0
	INSIGHT RemoteChainFormat = 1
	BULWARK RemoteChainFormat = 2
	CRYPTOID RemoteChainFormat = 3
	COINEXPLORER RemoteChainFormat = 4
	BLOCKBOOK RemoteChainFormat = 5
	RPC RemoteChainFormat = 6
)

type GetPeerInfoResponse struct {
	PossiblePeers []PossiblePeer
}

type PossiblePeer struct {
	Addr            string        `json:"addr"`
	Addrlocal       string        `json:"addrlocal"`
}

type RemoteChainFormat int

func StringToRemoteChain(format string, url string, username string, password string) RemoteChain {
	var remoteChain RemoteChain
	switch {
	case StringToRemoteChainFormat(format) == IQUIDUS:
		remoteChain = &IquidusExplorer{}
	case StringToRemoteChainFormat(format) == INSIGHT:
		remoteChain = &InsightExplorer{}
	case StringToRemoteChainFormat(format) == BULWARK:
		remoteChain = &BulwarkExplorer{}
	case StringToRemoteChainFormat(format) == CRYPTOID:
		remoteChain = &CryptoidExplorer{}
	case StringToRemoteChainFormat(format) == COINEXPLORER:
		remoteChain = &CoinExplorer{}
	case StringToRemoteChainFormat(format) == BLOCKBOOK:
		remoteChain = &BlockbookExplorer{}
	case StringToRemoteChainFormat(format) == RPC:
		remoteChain = &RPCExplorer{}
	}

	//configure it
	remoteChain.SetURL(url)
	remoteChain.SetUsername(username)
	remoteChain.SetPassword(password)

	return remoteChain
}

func StringToRemoteChainFormat(str string) RemoteChainFormat {
	str = strings.ToUpper(str)
	switch {
	case str == "IQUIDUS":
		return IQUIDUS
	case str == "INSIGHT":
		return INSIGHT
	case str == "BULWARK":
		return BULWARK
	case str == "CRYPTOID":
		return CRYPTOID
	case str == "COINEXPLORER":
		return COINEXPLORER
	case str == "BLOCKBOOK":
		return BLOCKBOOK
	case str == "RPC":
		return RPC
	}
	return UNKNOWN
}

type RemoteChainConnectionDefinition struct {
	Username		string  `json:username,omitempty`
	Password		string  `json:password,omitempty`
	Format          string 	`json:"format"`
	URL          	string 	`json:"url"`
}

type RemoteChainConnections []RemoteChainConnectionDefinition

func ParseRemoteChains(data string) (RemoteChainConnections, error) {
	var result RemoteChainConnections

	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		log.Fatal(err)
		return RemoteChainConnections{}, err
	}

	return result, nil
}

type RemoteChain interface {
	GetBlockHash(blockNumber int) (chainhash.Hash, error)
	GetPeers(portFilter uint32) ([]database.Peer, error)
	GetChainHeight() (int, error)
	GetTransaction(id string) (string, error)
	SetURL(url string)
	SetUsername(username string)
	SetPassword(password string)
}

func AddPossiblePeer(peer database.Peer, peers []database.Peer, portFilter uint32) []database.Peer {
	if peer.Port == portFilter {
		for _, prevPeer := range peers {
			if peer.Address == prevPeer.Address && peer.Port == prevPeer.Port {
				return peers
			}
		}
		return append(peers, peer)
	}
	return peers
}

func SplitAddress(pair string) (wire.NetAddress, error) {
	host, port, err := net.SplitHostPort(pair)
	if err != nil {
		log.Println(err)
	}

	parsedPort, err := strconv.Atoi(port)

	return wire.NetAddress{time.Now(),
		0,
		net.ParseIP(host),
		uint16(parsedPort)}, nil
}