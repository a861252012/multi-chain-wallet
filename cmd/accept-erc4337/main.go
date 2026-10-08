// accept-erc4337 checks one SimpleAccount deployment and transfer on Sepolia.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	aa "github.com/a861252012/multi-chain-wallet/internal/wallet/erc4337"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var factory = common.HexToAddress("0x9406Cc6185a346906296840746125a0E44976454")
var entryPoint = aa.EntryPointV06
var chainID = big.NewInt(11155111)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	dir := flag.String("dir", "data/erc4337-acceptance", "dedicated acceptance wallet and operation directory")
	rpcURL := flag.String("rpc", "https://ethereum-sepolia-rpc.publicnode.com", "Sepolia RPC")
	confirmRPC := flag.String("confirm-rpc", "", "required Sepolia RPC from an independent provider to confirm the funding address")
	bundlerURL := flag.String("bundler", "https://api.candide.dev/public/v3/11155111", "Sepolia bundler")
	send := flag.Bool("send", false, "sign and submit one operation; later runs reuse it")
	verify := flag.Bool("verify", false, "verify the saved operation without signing or sending")
	publicOwner := flag.String("owner", "", "public owner address for --verify without a keystore")
	flag.Parse()
	if err := validateConfirmationRPC(*rpcURL, *confirmRPC); err != nil {
		return err
	}
	if *send && *verify {
		return errors.New("choose --send or --verify")
	}
	if *publicOwner != "" && (!*verify || !common.IsHexAddress(*publicOwner)) {
		return errors.New("--owner requires --verify and a valid address")
	}
	if !*verify {
		if err := os.MkdirAll(*dir, 0700); err != nil {
			return err
		}
		lock, err := os.OpenFile(filepath.Join(*dir, "lock"), os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		defer lock.Close()
		if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			return errors.New("acceptance directory is in use")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	raw, err := rpc.DialContext(ctx, *rpcURL)
	if err != nil {
		return err
	}
	defer raw.Close()
	node := ethclient.NewClient(raw)
	id, err := node.ChainID(ctx)
	if err != nil {
		return err
	}
	if id.Cmp(chainID) != 0 {
		return errors.New("only Ethereum Sepolia is allowed")
	}
	brpc, err := rpc.DialContext(ctx, *bundlerURL)
	if err != nil {
		return err
	}
	defer brpc.Close()
	var bid string
	if err = brpc.CallContext(ctx, &bid, "eth_chainId"); err != nil {
		return err
	}
	if bid != "0xaa36a7" {
		return errors.New("bundler is not on Sepolia")
	}
	var supported []common.Address
	if err = brpc.CallContext(ctx, &supported, "eth_supportedEntryPoints"); err != nil {
		return err
	}
	found := false
	for _, a := range supported {
		found = found || a == entryPoint
	}
	if !found {
		return errors.New("bundler does not support EntryPoint v0.6")
	}
	for _, a := range []common.Address{factory, entryPoint} {
		code, e := node.CodeAt(ctx, a, nil)
		if e != nil {
			return e
		}
		if len(code) == 0 {
			return fmt.Errorf("missing contract: %s", a)
		}
	}
	km := wallet.NewKeystoreManager(*dir, 0, 0)
	if !km.Exists() && *publicOwner == "" {
		if *verify {
			return errors.New("no acceptance wallet")
		}
		password := make([]byte, 32)
		if _, err = rand.Read(password); err != nil {
			return err
		}
		p := []byte(hex.EncodeToString(password))
		if err = writeNew(filepath.Join(*dir, "password"), p); err != nil {
			return err
		}
		if _, err = km.Create(string(p)); err != nil {
			return err
		}
	}
	ownerString := *publicOwner
	if ownerString == "" {
		ownerString, err = km.Address()
		if err != nil {
			return err
		}
	}
	owner := common.HexToAddress(ownerString)
	args := append(common.LeftPadBytes(owner.Bytes(), 32), make([]byte, 32)...)
	confirmation, err := ethclient.DialContext(ctx, *confirmRPC)
	if err != nil {
		return errors.New("cannot connect to independent confirmation RPC")
	}
	defer confirmation.Close()
	sender, err := confirmedSender(ctx, node, confirmation, args)
	if err != nil {
		return err
	}
	balance, err := node.BalanceAt(ctx, sender, nil)
	if err != nil {
		return err
	}
	fmt.Printf("network=Sepolia entryPoint=%s factory=%s\nowner=%s sender=%s balanceWei=%s\n", entryPoint, factory, owner, sender, balance)
	bundler, err := aa.NewClient(*bundlerURL)
	if err != nil {
		return err
	}
	opFile := filepath.Join(*dir, "operation.json")
	var op aa.UserOperation
	data, err := os.ReadFile(opFile)
	if err == nil {
		if err = json.Unmarshal(data, &op); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	} else {
		if *verify {
			return errors.New("no saved operation")
		}
		if !*send {
			return nil
		}
		code, e := node.CodeAt(ctx, sender, nil)
		if e != nil {
			return e
		}
		if len(code) != 0 {
			return errors.New("use a fresh acceptance wallet: this account is already deployed")
		}
		recipientBalance, e := node.BalanceAt(ctx, owner, nil)
		if e != nil {
			return e
		}
		if recipientBalance.Sign() != 0 {
			return errors.New("acceptance recipient must start with zero balance")
		}
		tip, e := node.SuggestGasTipCap(ctx)
		if e != nil {
			return e
		}
		head, e := node.HeaderByNumber(ctx, nil)
		if e != nil {
			return e
		}
		initCode := append(factory.Bytes(), append(crypto.Keccak256([]byte("createAccount(address,uint256)"))[:4], args...)...)
		builder := aa.NewBuilder(entryPoint, chainID).SetSender(sender).SetNonce(big.NewInt(0)).SetInitCode(initCode).EstimateGasFees(head.BaseFee, tip).SetGasLimits(big.NewInt(0), big.NewInt(0), big.NewInt(0))
		if _, err = builder.SetExecuteCallData(owner, big.NewInt(1), nil); err != nil {
			return err
		}
		built, e := builder.Build()
		if e != nil {
			return e
		}
		op = *built
		p, e := os.ReadFile(filepath.Join(*dir, "password"))
		if e != nil {
			return e
		}
		signer, e := aa.NewKeystoreSigner(km, string(p))
		if e != nil {
			return e
		}
		op.Signature, e = aa.SignUserOpWithEthPrefix(signer, &op, entryPoint, chainID)
		if e != nil {
			return e
		}
		estimate, e := bundler.EstimateUserOperationGas(ctx, &op, entryPoint)
		if e != nil {
			return e
		}
		for _, v := range []*big.Int{estimate.CallGasLimit, estimate.VerificationGasLimit, estimate.PreVerificationGas} {
			if v == nil || v.Sign() <= 0 {
				return errors.New("invalid bundler gas estimate")
			}
			v.Mul(v, big.NewInt(12))
			v.Div(v, big.NewInt(10))
		}
		op.CallGasLimit = estimate.CallGasLimit
		op.VerificationGasLimit = estimate.VerificationGasLimit
		op.PreVerificationGas = estimate.PreVerificationGas
		// Extra reserve used in the successful v0.6 acceptance run, not a general fix.
		// It increases the charged pre-verification gas; the fee check below still applies.
		op.PreVerificationGas.Add(op.PreVerificationGas, op.VerificationGasLimit)
		cost := new(big.Int).Add(op.CallGasLimit, op.VerificationGasLimit)
		cost.Add(cost, op.PreVerificationGas)
		cost.Mul(cost, op.MaxFeePerGas)
		if cost.Cmp(big.NewInt(3_000_000_000_000_000)) > 0 {
			return fmt.Errorf("gas ceiling %s wei exceeds 0.003 test ETH", cost)
		}
		if balance.Cmp(new(big.Int).Add(cost, big.NewInt(1))) < 0 {
			return fmt.Errorf("fund sender first: need at most %s wei", new(big.Int).Add(cost, big.NewInt(1)))
		}
		op.Signature, e = aa.SignUserOpWithEthPrefix(signer, &op, entryPoint, chainID)
		if e != nil {
			return e
		}
		data, e = json.MarshalIndent(&op, "", "  ")
		if e != nil {
			return e
		}
		if e = writeNew(opFile, data); e != nil {
			return e
		}
	}
	if err = validateOperation(&op, sender, owner, args); err != nil {
		return err
	}
	hash, err := aa.GetUserOpHash(&op, entryPoint, chainID)
	if err != nil {
		return err
	}
	fmt.Printf("userOpHash=%s\n", hash)
	receipt, err := bundler.GetUserOperationReceipt(ctx, hash)
	if errors.Is(err, aa.ErrReceiptNotFound) && *send {
		got, e := bundler.SendUserOperation(ctx, &op, entryPoint)
		if e != nil {
			return e
		}
		if got != hash {
			return errors.New("bundler returned a different userOpHash")
		}
		fmt.Println("submitted; waiting for receipt")
		receipt, err = bundler.WaitForUserOperationReceipt(ctx, hash, 3*time.Second)
	}
	if err != nil {
		return err
	}
	if err = verifyReceipt(ctx, node, &op, owner, hash, receipt); err != nil {
		return err
	}
	evidence, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if !*verify {
		if err = os.WriteFile(filepath.Join(*dir, "receipt.json"), evidence, 0600); err != nil {
			return err
		}
	}
	fmt.Printf("verified: deployment, owner, EntryPoint event, transaction success, recipient +1 wei; tx=%s block=%s\n", receipt.Receipt.TxHash, receipt.Receipt.BlockNumber)
	return nil
}

func writeNew(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func validateOperation(op *aa.UserOperation, sender, owner common.Address, args []byte) error {
	if err := op.Validate(); err != nil {
		return err
	}
	b := aa.NewBuilder(entryPoint, chainID)
	if _, err := b.SetExecuteCallData(owner, big.NewInt(1), nil); err != nil {
		return err
	}
	expected, err := b.SetSender(sender).Build()
	if err != nil {
		return err
	}
	initCode := append(factory.Bytes(), append(crypto.Keccak256([]byte("createAccount(address,uint256)"))[:4], args...)...)
	if op.Sender != sender || op.Nonce.Sign() != 0 || len(op.PaymasterAndData) != 0 || !bytes.Equal(op.InitCode, initCode) || !bytes.Equal(op.CallData, expected.CallData) {
		return errors.New("saved operation does not match this acceptance")
	}
	hash, err := aa.GetUserOpHash(op, entryPoint, chainID)
	if err != nil {
		return err
	}
	valid, err := aa.VerifySignature(aa.EthSignedMessageHash(hash), op.Signature, owner)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid saved signature")
	}
	return nil
}

func verifyReceipt(ctx context.Context, node *ethclient.Client, op *aa.UserOperation, owner common.Address, hash common.Hash, r *aa.UserOperationReceipt) error {
	if r == nil || !r.Success || r.UserOpHash != hash || r.EntryPoint != entryPoint || r.Sender != op.Sender || r.Nonce == nil || r.Nonce.Cmp(op.Nonce) != 0 || r.Receipt == nil {
		return errors.New("UserOperation receipt mismatch or failure")
	}
	txr, err := node.TransactionReceipt(ctx, r.Receipt.TxHash)
	if err != nil {
		return err
	}
	if txr.Status != 1 || txr.TxHash != r.Receipt.TxHash || txr.BlockNumber == nil || txr.BlockNumber.Sign() == 0 || r.Receipt.BlockNumber == nil || txr.BlockNumber.Cmp(r.Receipt.BlockNumber) != 0 || txr.BlockHash != r.Receipt.BlockHash {
		return errors.New("underlying transaction failed or receipt mismatch")
	}
	head, err := node.HeaderByNumber(ctx, txr.BlockNumber)
	if err != nil {
		return err
	}
	if head.Hash() != txr.BlockHash {
		return errors.New("receipt block is no longer canonical")
	}
	topic := crypto.Keccak256Hash([]byte("UserOperationEvent(bytes32,address,address,uint256,bool,uint256,uint256)"))
	matched := false
	deployed := false
	deploymentTopic := crypto.Keccak256Hash([]byte("AccountDeployed(bytes32,address,address,address)"))
	for _, l := range txr.Logs {
		if l == nil {
			return errors.New("invalid receipt log")
		}
		if l.Address == entryPoint && len(l.Topics) == 3 && l.Topics[0] == deploymentTopic && l.Topics[1] == hash && l.Topics[2] == common.BytesToHash(op.Sender.Bytes()) && len(l.Data) == 64 && bytes.Equal(l.Data[:32], common.LeftPadBytes(factory.Bytes(), 32)) && new(big.Int).SetBytes(l.Data[32:]).Sign() == 0 && !l.Removed {
			deployed = true
		}
		if l.Address == entryPoint && len(l.Topics) == 4 && l.Topics[0] == topic && l.Topics[1] == hash && l.Topics[2] == common.BytesToHash(op.Sender.Bytes()) && l.Topics[3] == (common.Hash{}) && len(l.Data) == 128 && new(big.Int).SetBytes(l.Data[:32]).Cmp(op.Nonce) == 0 && new(big.Int).SetBytes(l.Data[32:64]).Cmp(big.NewInt(1)) == 0 && !l.Removed {
			matched = true
		}
	}
	if !matched {
		return errors.New("missing successful UserOperationEvent")
	}
	if !deployed {
		return errors.New("missing matching AccountDeployed event")
	}
	prior := new(big.Int).Sub(txr.BlockNumber, big.NewInt(1))
	oldCode, err := node.CodeAt(ctx, op.Sender, prior)
	if err != nil {
		return err
	}
	code, err := node.CodeAt(ctx, op.Sender, txr.BlockNumber)
	if err != nil {
		return err
	}
	if len(oldCode) != 0 || len(code) == 0 {
		return errors.New("account deployment not observed in receipt block")
	}
	for method, want := range map[string]common.Address{"owner()": owner, "entryPoint()": entryPoint} {
		got, e := node.CallContract(ctx, ethereum.CallMsg{To: &op.Sender, Data: crypto.Keccak256([]byte(method))[:4]}, txr.BlockNumber)
		if e != nil {
			return e
		}
		if len(got) != 32 || common.BytesToAddress(got) != want {
			return fmt.Errorf("account %s mismatch", method)
		}
	}
	before, err := node.BalanceAt(ctx, owner, prior)
	if err != nil {
		return err
	}
	after, err := node.BalanceAt(ctx, owner, txr.BlockNumber)
	if err != nil {
		return err
	}
	if before.Sign() != 0 || new(big.Int).Sub(after, before).Cmp(big.NewInt(1)) != 0 {
		return errors.New("recipient balance change is not exactly 1 wei")
	}
	head, err = node.HeaderByNumber(ctx, txr.BlockNumber)
	if err != nil {
		return err
	}
	if head.Hash() != txr.BlockHash {
		return errors.New("block changed during verification")
	}
	// Save the checked node receipt; this bundler omits the nested status field.
	r.Receipt = txr
	return nil
}

// Distinct hosts catch accidental reuse, not common ownership. The operator must
// choose independent providers; agreement is not a consensus or CREATE2 proof.
func validateConfirmationRPC(primary, confirmation string) error {
	a, errA := url.Parse(primary)
	b, errB := url.Parse(confirmation)
	if errA != nil || errB != nil || a.Hostname() == "" || b.Hostname() == "" ||
		(a.Scheme != "https" && a.Scheme != "http") || (b.Scheme != "https" && b.Scheme != "http") {
		return errors.New("--rpc and --confirm-rpc must be HTTP(S) endpoints from independent providers")
	}
	normalize := func(host string) string {
		host = strings.ToLower(strings.TrimSuffix(host, "."))
		if ip := net.ParseIP(host); ip != nil {
			if ip.IsLoopback() {
				return "localhost"
			}
			return ip.String()
		}
		return host
	}
	if normalize(a.Hostname()) == normalize(b.Hostname()) {
		return errors.New("confirmation RPC must use an independent provider, not the same host")
	}
	return nil
}

func confirmedSender(ctx context.Context, primary, confirmation *ethclient.Client, args []byte) (common.Address, error) {
	var sender common.Address
	for i, node := range []*ethclient.Client{primary, confirmation} {
		id, err := node.ChainID(ctx)
		if err != nil || id.Cmp(chainID) != 0 {
			return common.Address{}, errors.New("funding address confirmation requires two Sepolia RPCs")
		}
		result, err := node.CallContract(ctx, ethereum.CallMsg{To: &factory, Data: append(crypto.Keccak256([]byte("getAddress(address,uint256)"))[:4], args...)}, nil)
		if err != nil {
			return common.Address{}, errors.New("cannot confirm factory address with both RPCs")
		}
		if len(result) != 32 || !bytes.Equal(result[:12], make([]byte, 12)) || common.BytesToAddress(result) == (common.Address{}) {
			return common.Address{}, errors.New("invalid factory address response")
		}
		address := common.BytesToAddress(result)
		if i == 0 {
			sender = address
		} else if sender != address {
			return common.Address{}, errors.New("RPCs disagree on the funding address; do not transfer funds")
		}
	}
	return sender, nil
}
