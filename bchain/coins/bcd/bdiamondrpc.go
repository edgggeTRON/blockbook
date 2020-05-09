package bcd

import (
	"blockbook/bchain"
	"blockbook/bchain/coins/btc"
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/juju/errors"
)

// BdiamondRPC is an interface to JSON-RPC bitcoind service
type BdiamondRPC struct {
	*btc.BitcoinRPC
}

// NewBdiamondRPC returns new BdiamondRPC instance
func NewBdiamondRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}
	z := &BdiamondRPC{
		BitcoinRPC: b.(*btc.BitcoinRPC),
	}
	fmt.Println("NewBdiamondRPC:", z)
	z.RPCMarshaler = btc.JSONMarshalerV1{}
	z.ChainConfig.SupportsEstimateFee = false
	z.ChainConfig.SupportsEstimateSmartFee = true
	return z, nil
}

// Initialize initializes BdiamondRPC instance
func (z *BdiamondRPC) Initialize() error {
	ci, err := z.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	params := GetChainParams(chainName)

	z.Parser = NewBdiamondParser(params, z.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		z.Testnet = false
		z.Network = "livenet"
	} else {
		z.Testnet = true
		z.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

// GetBlock returns block with given hash.
func (z *BdiamondRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" && height > 0 {
		hash, err = z.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}

	glog.V(1).Info("rpc: getblock (verbosity=1) ", hash)

	res := btc.ResGetBlockThin{}
	req := btc.CmdGetBlock{Method: "getblock"}
	req.Params.BlockHash = hash
	req.Params.Verbosity = 1
	err = z.Call(&req, &res)

	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	if res.Error != nil {
		return nil, errors.Annotatef(res.Error, "hash %v", hash)
	}

	txs := make([]bchain.Tx, 0, len(res.Result.Txids))
	for _, txid := range res.Result.Txids {
		tx, err := z.GetTransaction(txid)
		if err != nil {
			if err == bchain.ErrTxNotFound {
				glog.Errorf("rpc: getblock: skipping transanction in block %s due error: %s", hash, err)
				continue
			}
			return nil, err
		}
		txs = append(txs, *tx)
	}
	block := &bchain.Block{
		BlockHeader: res.Result.BlockHeader,
		Txs:         txs,
	}
	return block, nil
}

// GetTransactionForMempool returns a transaction by the transaction ID.
// It could be optimized for mempool, i.e. without block time and confirmations
func (z *BdiamondRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
	fmt.Println(txid)
	return z.GetTransaction(txid)
}

// GetMempoolEntry returns mempool data for given transaction
func (z *BdiamondRPC) GetMempoolEntry(txid string) (*bchain.MempoolEntry, error) {
	return nil, errors.New("GetMempoolEntry: not implemented")
}

func isErrBlockNotFound(err *bchain.RPCError) bool {
	return err.Message == "Block not found" ||
		err.Message == "Block height out of range"
}
