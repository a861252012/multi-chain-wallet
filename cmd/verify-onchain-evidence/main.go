// Command verify-onchain-evidence rechecks recorded evidence with read-only RPCs.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
)

type network struct {
	chainID  int64
	endpoint string
}

var networks = map[string]network{
	"sepolia":  {chain.SepoliaID, "https://ethereum-sepolia-rpc.publicnode.com"},
	"arbitrum": {chain.ArbitrumSepoliaID, "https://sepolia-rollup.arbitrum.io/rpc"},
	"base":     {chain.BaseSepoliaID, "https://sepolia.base.org"},
	"optimism": {chain.OptimismSepoliaID, "https://sepolia.optimism.io"},
	"polygon":  {chain.PolygonAmoyID, "https://polygon-amoy.drpc.org"},
}

const tronEndpoint = "https://api.shasta.trongrid.io"
const genesis = "0000000000000000de1aa88295e1fcf982742f773e0419c5a9c134c994a9059e"

type evidence struct {
	Network       string `json:"network"`
	ChainID       int64  `json:"chainId"`
	Hash          string `json:"hash"`
	From          string `json:"from"`
	To            string `json:"to"`
	Operation     string `json:"operation"`
	AmountRaw     string `json:"amountRaw"`
	Contract      string `json:"contract"`
	TxTo          string `json:"txTo"`
	Calldata      string `json:"calldata"`
	TokenOut      string `json:"tokenOut"`
	MinimumOutRaw string `json:"minimumOutRaw"`
}
type result struct {
	Network   string `json:"network"`
	Hash      string `json:"hash"`
	Block     uint64 `json:"block"`
	Finalized bool   `json:"finalized"`
	FeeSun    *int64 `json:"feeSun,omitempty"`
}
type verifier struct{ client *http.Client }

func (v verifier) post(endpoint string, body, target any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	res, err := v.client.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("HTTP status %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(target)
}
func (v verifier) rpc(endpoint, method string, params []any, target any) error {
	var response struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	err := v.post(endpoint, struct {
		Version string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
		Params  []any  `json:"params"`
	}{"2.0", 1, method, params}, &response)
	if err != nil {
		return err
	}
	if len(response.Error) > 0 && string(response.Error) != "null" {
		return fmt.Errorf("RPC failed: %s", method)
	}
	return json.Unmarshal(response.Result, target)
}
func quantity(s string) (*big.Int, error) {
	if !strings.HasPrefix(s, "0x") {
		return nil, errors.New("invalid RPC quantity")
	}
	n, ok := new(big.Int).SetString(s[2:], 16)
	if !ok || n.Sign() < 0 {
		return nil, errors.New("invalid RPC quantity")
	}
	return n, nil
}
func (v verifier) verify(row evidence) (result, error) {
	out := result{Network: row.Network, Hash: row.Hash}
	if row.Network == "tron" {
		return v.verifyTron(row)
	}
	net, ok := networks[row.Network]
	if !ok {
		return out, errors.New("unknown network")
	}
	if row.ChainID != net.chainID || !regexp.MustCompile(`^0x[0-9a-f]{64}$`).MatchString(row.Hash) {
		return out, errors.New("invalid EVM network or transaction ID")
	}
	var chain string
	if err := v.rpc(net.endpoint, "eth_chainId", []any{}, &chain); err != nil {
		return out, err
	}
	id, err := quantity(chain)
	if err != nil || !id.IsInt64() || id.Int64() != net.chainID {
		return out, errors.New("wrong EVM network")
	}
	var receipt struct {
		Hash        string `json:"transactionHash"`
		Status      string `json:"status"`
		BlockNumber string `json:"blockNumber"`
		BlockHash   string `json:"blockHash"`
		Logs        []struct {
			Address string   `json:"address"`
			Data    string   `json:"data"`
			Topics  []string `json:"topics"`
		} `json:"logs"`
	}
	if err := v.rpc(net.endpoint, "eth_getTransactionReceipt", []any{row.Hash}, &receipt); err != nil {
		return out, err
	}
	if receipt.Hash != row.Hash || receipt.Status != "0x1" {
		return out, errors.New("no successful receipt")
	}
	var tx struct{ Hash, From, To, Value, Input, ChainID, BlockHash string }
	if err := v.rpc(net.endpoint, "eth_getTransactionByHash", []any{row.Hash}, &tx); err != nil {
		return out, err
	}
	if tx.Hash != row.Hash || !strings.EqualFold(tx.From, row.From) {
		return out, errors.New("sender/transaction mismatch")
	}
	if row.TxTo != "" && !strings.EqualFold(tx.To, row.TxTo) {
		return out, errors.New("actual transaction target mismatch")
	}
	if row.Calldata != "" && !strings.EqualFold(tx.Input, row.Calldata) {
		return out, errors.New("signed calldata mismatch")
	}
	switch row.Operation {
	case "native", "base", "optimism", "arbitrum", "polygon", "wrap":
		value, e := quantity(tx.Value)
		amount, valid := new(big.Int).SetString(row.AmountRaw, 10)
		if e != nil || !valid || amount.Sign() < 0 || value.Cmp(amount) != 0 {
			return out, errors.New("native amount mismatch")
		}
	case "swap", "swap-back":
		if len(row.From) != 42 {
			return out, errors.New("invalid swap recipient")
		}
		recipient := "0x" + strings.Repeat("0", 24) + strings.ToLower(row.From[2:])
		output := new(big.Int)
		for _, log := range receipt.Logs {
			if strings.EqualFold(log.Address, row.TokenOut) && len(log.Topics) == 3 && log.Topics[0] == "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" && strings.EqualFold(log.Topics[2], recipient) {
				value, e := quantity(log.Data)
				if e != nil {
					return out, e
				}
				output.Add(output, value)
			}
		}
		minimum, valid := new(big.Int).SetString(row.MinimumOutRaw, 10)
		if !valid || minimum.Sign() < 0 || output.Cmp(minimum) < 0 {
			return out, errors.New("swap output logs below bound minimum")
		}
	}
	signedID, e := quantity(tx.ChainID)
	if e != nil || !signedID.IsInt64() || signedID.Int64() != net.chainID || tx.BlockHash != receipt.BlockHash {
		return out, errors.New("signed network or inclusion mismatch")
	}
	var block struct {
		Hash         string
		Transactions []string
	}
	if err := v.rpc(net.endpoint, "eth_getBlockByNumber", []any{receipt.BlockNumber, false}, &block); err != nil {
		return out, err
	}
	if block.Hash != receipt.BlockHash || !slices.Contains(block.Transactions, row.Hash) {
		return out, errors.New("receipt block is not canonical")
	}
	number, e := quantity(receipt.BlockNumber)
	if e != nil || !number.IsUint64() {
		return out, errors.New("invalid receipt block")
	}
	out.Block = number.Uint64()
	var final *struct{ Number string }
	if err := v.rpc(net.endpoint, "eth_getBlockByNumber", []any{"finalized", false}, &final); err != nil {
		return out, err
	}
	if final != nil {
		n, e := quantity(final.Number)
		if e != nil {
			return out, e
		}
		out.Finalized = n.Cmp(number) >= 0
	}
	return out, nil
}
func (v verifier) verifyTron(row evidence) (result, error) {
	out := result{Network: row.Network, Hash: row.Hash}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(row.Hash) {
		return out, errors.New("invalid TRON transaction ID")
	}
	var block struct{ BlockID string }
	if err := v.post(tronEndpoint+"/wallet/getblockbynum", struct {
		Num int `json:"num"`
	}{0}, &block); err != nil {
		return out, err
	}
	if block.BlockID != genesis {
		return out, errors.New("wrong TRON network")
	}
	var receipt struct {
		ID             string
		BlockNumber    uint64
		Result         string
		Receipt        struct{ Result string }
		ContractResult []string
		Fee            int64
	}
	if err := v.post(tronEndpoint+"/walletsolidity/gettransactioninfobyid", struct {
		Value string `json:"value"`
	}{row.Hash}, &receipt); err != nil {
		return out, err
	}
	if receipt.ID != row.Hash || receipt.BlockNumber == 0 {
		return out, errors.New("no solidified receipt")
	}
	if receipt.Result == "FAILED" || (receipt.Receipt.Result != "" && receipt.Receipt.Result != "SUCCESS") {
		return out, errors.New("TRON execution failed")
	}
	if row.Contract != "" && !slices.Equal(receipt.ContractResult, []string{strings.Repeat("0", 63) + "1"}) {
		return out, errors.New("TRC-20 did not return true")
	}
	var tx struct {
		TxID    string
		RawData struct {
			Contract []struct {
				Parameter struct {
					Value struct {
						Owner    string `json:"owner_address"`
						To       string `json:"to_address"`
						Contract string `json:"contract_address"`
					}
				}
			}
		} `json:"raw_data"`
	}
	if err := v.post(tronEndpoint+"/walletsolidity/gettransactionbyid", struct {
		Value   string `json:"value"`
		Visible bool   `json:"visible"`
	}{row.Hash, true}, &tx); err != nil {
		return out, err
	}
	if tx.TxID != row.Hash || len(tx.RawData.Contract) == 0 {
		return out, errors.New("TRON transaction mismatch")
	}
	value := tx.RawData.Contract[0].Parameter.Value
	if value.Owner != row.From {
		return out, errors.New("TRON sender mismatch")
	}
	if row.Contract != "" {
		if value.Contract != row.Contract {
			return out, errors.New("TRC-20 contract mismatch")
		}
	} else if value.To != row.To {
		return out, errors.New("TRX recipient mismatch")
	}
	out.Block = receipt.BlockNumber
	out.Finalized = true
	out.FeeSun = &receipt.Fee
	return out, nil
}
func run(args []string, client *http.Client, output io.Writer) error {
	flags := flag.NewFlagSet("verify-onchain-evidence", flag.ContinueOnError)
	flags.SetOutput(output)
	filter := flags.String("network", "", "Filter: sepolia, arbitrum, base, optimism, polygon, tron")
	path := flags.String("manifest", "internal/web/static/onchain-evidence.json", "Evidence manifest path (relative to working directory)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if _, ok := networks[*filter]; *filter != "" && *filter != "tron" && !ok {
		return errors.New("unknown network")
	}
	data, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var manifest struct{ Transactions []evidence }
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	v := verifier{client}
	count, failures := 0, 0
	encoder := json.NewEncoder(output)
	for _, row := range manifest.Transactions {
		if *filter != "" && row.Network != *filter {
			continue
		}
		count++
		result, err := v.verify(row)
		if err != nil {
			failures++
			if e := encoder.Encode(struct {
				Network string `json:"network"`
				Hash    string `json:"hash"`
				Error   string `json:"error"`
			}{row.Network, row.Hash, err.Error()}); e != nil {
				return e
			}
		} else if err := encoder.Encode(result); err != nil {
			return err
		}
	}
	if count == 0 {
		return errors.New("no published outgoing evidence for this network; acceptance is incomplete")
	}
	if failures > 0 {
		return fmt.Errorf("%d evidence checks failed; do not report full acceptance", failures)
	}
	fmt.Fprintf(output, "PASS: %d receipts independently rechecked; no signing or broadcast.\n", count)
	return nil
}
func main() {
	if err := run(os.Args[1:], &http.Client{Timeout: 25 * time.Second}, os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
