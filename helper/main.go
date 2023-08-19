package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ipfsApiAddr        = "http://127.0.0.1:5001"
	bscRpcAddr         = "https://rpc-bsc.48.club"
	nftAddr            = common.HexToAddress("0x57b81C140BdfD35dbfbB395360a66D54a650666D")
	nftAbi, _          = abi.JSON(strings.NewReader(`[{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"_tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"}]`))
	totalSupplyData, _ = nftAbi.Pack("totalSupply")
	mcAddr             = common.HexToAddress("0x41263cBA59EB80dC200F3E2544eda4ed6A90E76C")
	mcAbi, _           = abi.JSON(strings.NewReader(`[{"constant":false,"inputs":[{"components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}],"name":"calls","type":"tuple[]"}],"name":"aggregate","outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`))
)

func main() {
	tc := time.NewTicker(1 * time.Minute)
	for {
		infoIpfs := formatIpfs(getIpfsHash())
		pinIpfs(infoIpfs)
		imgIpfs := formatIpfs(getInfo(infoIpfs))
		pinIpfs(imgIpfs)
		<-tc.C
	}
}

func getInfo(ipfs []string) (newList []string) {
	for _, v := range ipfs {
		resp, err := http.Post(fmt.Sprintf("%s/api/v0/cat?arg=%s", ipfsApiAddr, v), "text/plain; charset=utf-8", nil)
		if resp.StatusCode != http.StatusOK && err != nil {
			log.Printf("ipfs cat err: %s", err.Error())
			continue
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("ipfs cat err: %s", err.Error())
			continue
		}
		type Image struct {
			Image string `json:"image"`
		}
		var image Image
		_ = json.Unmarshal(b, &image)
		if image.Image != "" {
			newList = append(newList, image.Image)
		}
	}
	return
}

func pinIpfs(ipfs []string) {
	for _, v := range ipfs {
		resp, err := http.Post(fmt.Sprintf("%s/api/v0/pin/add?arg=%s", ipfsApiAddr, v), "text/plain; charset=utf-8", nil)
		if err != nil {
			log.Printf("ipfs pin err: %s", err.Error())
		}
		defer resp.Body.Close()
	}
}

func formatIpfs(ipfs []string) []string {
	for k := range ipfs {
		ipfs[k] = strings.Replace(ipfs[k], "ipfs://", "/ipfs/", -1)
	}
	m := make(map[string]bool)
	for _, v := range ipfs {
		m[v] = true
	}
	ipfs = []string{}
	for k := range m {
		ipfs = append(ipfs, k)
	}
	return ipfs
}

func getIpfsHash() (ipfs []string) {
	clinet, err := ethclient.Dial(bscRpcAddr)
	if err != nil {
		log.Printf("rpc client err: %s", err.Error())
		return
	}

	type aggregate struct {
		Target   common.Address
		CallData []byte
	}
	tokenURIDatas := []aggregate{}
	for k := range make([]int, totalSupply(clinet).Uint64()) {
		tokenURIData, _ := nftAbi.Pack("tokenURI", big.NewInt(int64(k)))
		tokenURIDatas = append(tokenURIDatas, aggregate{Target: nftAddr, CallData: tokenURIData})
	}

	aggregateData, _ := mcAbi.Pack("aggregate", tokenURIDatas)
	aggregateHex, err := clinet.CallContract(context.Background(), ethereum.CallMsg{To: &mcAddr, Data: aggregateData}, nil)
	if err != nil {
		log.Printf("aggregate err: %s", err.Error())
	}

	aggregateList, _ := mcAbi.Unpack("aggregate", aggregateHex)

	for _, v := range aggregateList[1].([][]uint8) {
		tokenURI, _ := nftAbi.Unpack("tokenURI", v)
		ipfs = append(ipfs, tokenURI[0].(string))
	}

	return
}

func totalSupply(clinet *ethclient.Client) *big.Int {
	totalSupplyHex, err := clinet.CallContract(context.Background(), ethereum.CallMsg{To: &nftAddr, Data: totalSupplyData}, nil)
	if err != nil {
		log.Printf("rpc client err: %s", err.Error())
		return totalSupply(clinet)
	}

	totalSupplyList, _ := nftAbi.Unpack("totalSupply", totalSupplyHex)
	return totalSupplyList[0].(*big.Int)
}
