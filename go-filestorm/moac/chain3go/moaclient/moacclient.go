// Copyright 2019 The go-filestorm Authors
// This file is part of the go-filestorm library.
//
// The go-filestorm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-filestorm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-filestorm library. If not, see <http://www.gnu.org/licenses/>.

// Package moacclient provides a client for the Filestorm RPC API.
package moacclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/filestorm/go-filestorm"
	"github.com/filestorm/go-filestorm/common"
	"github.com/filestorm/go-filestorm/common/hexutil"
	"github.com/filestorm/go-filestorm/core/types"
	"github.com/filestorm/go-filestorm/rlp"
	"github.com/filestorm/go-filestorm/rpc"
)

// Client defines typed wrappers for the Filestorm RPC API.
type Client struct {
	c *rpc.Client
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *Client {
	return &Client{c}
}

func (mc *Client) Close() {
	mc.c.Close()
}

// Blockchain Access

// ChainId retrieves the current chain ID for transaction replay protection.
func (mc *Client) ChainID(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := mc.c.CallContext(ctx, &result, "mc_chainId")
	if err != nil {
		return nil, err
	}
	return (*big.Int)(&result), err
}

// BlockByHash returns the given full block.
//
// Note that loading full blocks requires two requests. Use HeaderByHash
// if you don't need all transactions or uncle headers.
func (mc *Client) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return mc.getBlock(ctx, "mc_getBlockByHash", hash, true)
}

// BlockByNumber returns a block from the current canonical chain. If number is nil, the
// latest known block is returned.
//
// Note that loading full blocks requires two requests. Use HeaderByNumber
// if you don't need all transactions or uncle headers.
func (mc *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return mc.getBlock(ctx, "mc_getBlockByNumber", toBlockNumArg(number), true)
}

type rpcBlock struct {
	Hash         common.Hash      `json:"hash"`
	Transactions []rpcTransaction `json:"transactions"`
	UncleHashes  []common.Hash    `json:"uncles"`
}

func (mc *Client) getBlock(ctx context.Context, method string, args ...interface{}) (*types.Block, error) {
	var raw json.RawMessage
	err := mc.c.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, filestorm.NotFound
	}
	// Decode header and transactions.
	var head *types.Header
	var body rpcBlock
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	// Quick-verify transaction and uncle lists. This mostly helps with debugging the server.
	if head.UncleHash == types.EmptyUncleHash && len(body.UncleHashes) > 0 {
		return nil, fmt.Errorf("server returned non-empty uncle list but block header indicates no uncles")
	}
	if head.UncleHash != types.EmptyUncleHash && len(body.UncleHashes) == 0 {
		return nil, fmt.Errorf("server returned empty uncle list but block header indicates uncles")
	}
	if head.TxHash == types.EmptyRootHash && len(body.Transactions) > 0 {
		return nil, fmt.Errorf("server returned non-empty transaction list but block header indicates no transactions")
	}
	if head.TxHash != types.EmptyRootHash && len(body.Transactions) == 0 {
		return nil, fmt.Errorf("server returned empty transaction list but block header indicates transactions")
	}
	// Load uncles because they are not included in the block response.
	var uncles []*types.Header
	if len(body.UncleHashes) > 0 {
		uncles = make([]*types.Header, len(body.UncleHashes))
		reqs := make([]rpc.BatchElem, len(body.UncleHashes))
		for i := range reqs {
			reqs[i] = rpc.BatchElem{
				Method: "mc_getUncleByBlockHashAndIndex",
				Args:   []interface{}{body.Hash, hexutil.EncodeUint64(uint64(i))},
				Result: &uncles[i],
			}
		}
		if err := mc.c.BatchCallContext(ctx, reqs); err != nil {
			return nil, err
		}
		for i := range reqs {
			if reqs[i].Error != nil {
				return nil, reqs[i].Error
			}
			if uncles[i] == nil {
				return nil, fmt.Errorf("got null header for uncle %d of block %x", i, body.Hash[:])
			}
		}
	}
	// Fill the sender cache of transactions in the block.
	txs := make([]*types.Transaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		if tx.From != nil {
			setSenderFromServer(tx.tx, *tx.From, body.Hash)
		}
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(head).WithBody(txs, uncles), nil
}

// HeaderByHash returns the block header with the given hash.
func (mc *Client) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	var head *types.Header
	err := mc.c.CallContext(ctx, &head, "mc_getBlockByHash", hash, false)
	if err == nil && head == nil {
		err = filestorm.NotFound
	}
	return head, err
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (mc *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var head *types.Header
	err := mc.c.CallContext(ctx, &head, "mc_getBlockByNumber", toBlockNumArg(number), false)
	if err == nil && head == nil {
		err = filestorm.NotFound
	}
	return head, err
}

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

// TransactionByHash returns the transaction with the given hash.
func (mc *Client) TransactionByHash(ctx context.Context, hash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	var json *rpcTransaction
	err = mc.c.CallContext(ctx, &json, "mc_getTransactionByHash", hash)
	if err != nil {
		return nil, false, err
	} else if json == nil {
		return nil, false, filestorm.NotFound
	} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
		return nil, false, fmt.Errorf("server returned transaction without signature")
	}
	if json.From != nil && json.BlockHash != nil {
		setSenderFromServer(json.tx, *json.From, *json.BlockHash)
	}
	return json.tx, json.BlockNumber == nil, nil
}

// TransactionSender returns the sender address of the given transaction. The transaction
// must be known to the remote node and included in the blockchain at the given block and
// index. The sender is the one derived by the protocol at the time of inclusion.
//
// There is a fast-path for transactions retrieved by TransactionByHash and
// TransactionInBlock. Getting their sender address can be done without an RPC interaction.
func (mc *Client) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	// Try to load the address from the cache.
	sender, err := types.Sender(&senderFromServer{blockhash: block}, tx)
	if err == nil {
		return sender, nil
	}
	var meta struct {
		Hash common.Hash
		From common.Address
	}
	if err = mc.c.CallContext(ctx, &meta, "mc_getTransactionByBlockHashAndIndex", block, hexutil.Uint64(index)); err != nil {
		return common.Address{}, err
	}
	if meta.Hash == (common.Hash{}) || meta.Hash != tx.Hash() {
		return common.Address{}, errors.New("wrong inclusion block/index")
	}
	return meta.From, nil
}

// TransactionCount returns the total number of transactions in the given block.
func (mc *Client) TransactionCount(ctx context.Context, blockHash common.Hash) (uint, error) {
	var num hexutil.Uint
	err := mc.c.CallContext(ctx, &num, "mc_getBlockTransactionCountByHash", blockHash)
	return uint(num), err
}

// TransactionInBlock returns a single transaction at index in the given block.
func (mc *Client) TransactionInBlock(ctx context.Context, blockHash common.Hash, index uint) (*types.Transaction, error) {
	var json *rpcTransaction
	err := mc.c.CallContext(ctx, &json, "mc_getTransactionByBlockHashAndIndex", blockHash, hexutil.Uint64(index))
	if err != nil {
		return nil, err
	}
	if json == nil {
		return nil, filestorm.NotFound
	} else if _, r, _ := json.tx.RawSignatureValues(); r == nil {
		return nil, fmt.Errorf("server returned transaction without signature")
	}
	if json.From != nil && json.BlockHash != nil {
		setSenderFromServer(json.tx, *json.From, *json.BlockHash)
	}
	return json.tx, err
}

// TransactionReceipt returns the receipt of a transaction by transaction hash.
// Note that the receipt is not available for pending transactions.
func (mc *Client) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	var r *types.Receipt
	err := mc.c.CallContext(ctx, &r, "mc_getTransactionReceipt", txHash)
	if err == nil {
		if r == nil {
			return nil, filestorm.NotFound
		}
	}
	return r, err
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}

type rpcProgress struct {
	StartingBlock hexutil.Uint64
	CurrentBlock  hexutil.Uint64
	HighestBlock  hexutil.Uint64
	PulledStates  hexutil.Uint64
	KnownStates   hexutil.Uint64
}

// SyncProgress retrieves the current progress of the sync algorithm. If there's
// no sync currently running, it returns nil.
func (mc *Client) SyncProgress(ctx context.Context) (*filestorm.SyncProgress, error) {
	var raw json.RawMessage
	if err := mc.c.CallContext(ctx, &raw, "mc_syncing"); err != nil {
		return nil, err
	}
	// Handle the possible response types
	var syncing bool
	if err := json.Unmarshal(raw, &syncing); err == nil {
		return nil, nil // Not syncing (always false)
	}
	var progress *rpcProgress
	if err := json.Unmarshal(raw, &progress); err != nil {
		return nil, err
	}
	return &filestorm.SyncProgress{
		StartingBlock: uint64(progress.StartingBlock),
		CurrentBlock:  uint64(progress.CurrentBlock),
		HighestBlock:  uint64(progress.HighestBlock),
		PulledStates:  uint64(progress.PulledStates),
		KnownStates:   uint64(progress.KnownStates),
	}, nil
}

// SubscribeNewHead subscribes to notifications about the current blockchain head
// on the given channel.
func (mc *Client) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (filestorm.Subscription, error) {
	return mc.c.EthSubscribe(ctx, ch, "newHeads")
}

// State Access

// NetworkID returns the network ID (also known as the chain ID) for this chain.
func (mc *Client) NetworkID(ctx context.Context) (*big.Int, error) {
	version := new(big.Int)
	var ver string
	if err := mc.c.CallContext(ctx, &ver, "net_version"); err != nil {
		return nil, err
	}
	if _, ok := version.SetString(ver, 10); !ok {
		return nil, fmt.Errorf("invalid net_version result %q", ver)
	}
	return version, nil
}

// BalanceAt returns the wei balance of the given account.
// The block number can be nil, in which case the balance is taken from the latest known block.
func (mc *Client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var result hexutil.Big
	err := mc.c.CallContext(ctx, &result, "mc_getBalance", account, toBlockNumArg(blockNumber))
	return (*big.Int)(&result), err
}

// StorageAt returns the value of key in the contract storage of the given account.
// The block number can be nil, in which case the value is taken from the latest known block.
func (mc *Client) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := mc.c.CallContext(ctx, &result, "mc_getStorageAt", account, key, toBlockNumArg(blockNumber))
	return result, err
}

// CodeAt returns the contract code of the given account.
// The block number can be nil, in which case the code is taken from the latest known block.
func (mc *Client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	var result hexutil.Bytes
	err := mc.c.CallContext(ctx, &result, "mc_getCode", account, toBlockNumArg(blockNumber))
	return result, err
}

// NonceAt returns the account nonce of the given account.
// The block number can be nil, in which case the nonce is taken from the latest known block.
func (mc *Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	var result hexutil.Uint64
	err := mc.c.CallContext(ctx, &result, "mc_getTransactionCount", account, toBlockNumArg(blockNumber))
	return uint64(result), err
}

// Filters

// FilterLogs executes a filter query.
func (mc *Client) FilterLogs(ctx context.Context, q filestorm.FilterQuery) ([]types.Log, error) {
	var result []types.Log
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	err = mc.c.CallContext(ctx, &result, "mc_getLogs", arg)
	return result, err
}

// SubscribeFilterLogs subscribes to the results of a streaming filter query.
func (mc *Client) SubscribeFilterLogs(ctx context.Context, q filestorm.FilterQuery, ch chan<- types.Log) (filestorm.Subscription, error) {
	arg, err := toFilterArg(q)
	if err != nil {
		return nil, err
	}
	return mc.c.EthSubscribe(ctx, ch, "logs", arg)
}

func toFilterArg(q filestorm.FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{
		"address": q.Addresses,
		"topics":  q.Topics,
	}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock == nil {
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = toBlockNumArg(q.FromBlock)
		}
		arg["toBlock"] = toBlockNumArg(q.ToBlock)
	}
	return arg, nil
}

// Pending State

// PendingBalanceAt returns the wei balance of the given account in the pending state.
func (mc *Client) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	var result hexutil.Big
	err := mc.c.CallContext(ctx, &result, "mc_getBalance", account, "pending")
	return (*big.Int)(&result), err
}

// PendingStorageAt returns the value of key in the contract storage of the given account in the pending state.
func (mc *Client) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	var result hexutil.Bytes
	err := mc.c.CallContext(ctx, &result, "mc_getStorageAt", account, key, "pending")
	return result, err
}

// PendingCodeAt returns the contract code of the given account in the pending state.
func (mc *Client) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	var result hexutil.Bytes
	err := mc.c.CallContext(ctx, &result, "mc_getCode", account, "pending")
	return result, err
}

// PendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (mc *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result hexutil.Uint64
	err := mc.c.CallContext(ctx, &result, "mc_getTransactionCount", account, "pending")
	return uint64(result), err
}

// PendingTransactionCount returns the total number of transactions in the pending state.
func (mc *Client) PendingTransactionCount(ctx context.Context) (uint, error) {
	var num hexutil.Uint
	err := mc.c.CallContext(ctx, &num, "mc_getBlockTransactionCountByNumber", "pending")
	return uint(num), err
}

// TODO: SubscribePendingTransactions (needs server side)

// Contract Calling

// CallContract executes a message call transaction, which is directly executed in the VM
// of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be nil, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
func (mc *Client) CallContract(ctx context.Context, msg filestorm.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var hex hexutil.Bytes
	err := mc.c.CallContext(ctx, &hex, "mc_call", toCallArg(msg), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return hex, nil
}

// PendingCallContract executes a message call transaction using the EVM.
// The state seen by the contract call is the pending state.
func (mc *Client) PendingCallContract(ctx context.Context, msg filestorm.CallMsg) ([]byte, error) {
	var hex hexutil.Bytes
	err := mc.c.CallContext(ctx, &hex, "mc_call", toCallArg(msg), "pending")
	if err != nil {
		return nil, err
	}
	return hex, nil
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (mc *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := mc.c.CallContext(ctx, &hex, "mc_gasPrice"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

// EstimateGas tries to estimate the gas needed to execute a specific transaction based on
// the current pending state of the backend blockchain. There is no guarantee that this is
// the true gas limit requirement as other transactions may be added or removed by miners,
// but it should provide a basis for setting a reasonable default.
func (mc *Client) EstimateGas(ctx context.Context, msg filestorm.CallMsg) (uint64, error) {
	var hex hexutil.Uint64
	err := mc.c.CallContext(ctx, &hex, "mc_estimateGas", toCallArg(msg))
	if err != nil {
		return 0, err
	}
	return uint64(hex), nil
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func (mc *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}
	return mc.c.CallContext(ctx, nil, "mc_sendRawTransaction", hexutil.Encode(data))
}

func toCallArg(msg filestorm.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}