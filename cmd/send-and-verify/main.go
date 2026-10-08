// Command send-and-verify performs explicit local Sepolia acceptance checks.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/term"
)

type response struct {
	ChainID int64   `json:"chainId"`
	Exists  bool    `json:"exists"`
	Address string  `json:"address"`
	CSRF    string  `json:"csrfToken"`
	Wei     *string `json:"wei"`
	ID      string  `json:"id"`
	Error   string  `json:"error"`
	To      string  `json:"to"`
	Amount  string  `json:"amount"`
	Total   string  `json:"totalEth"`
	Hash    string  `json:"hash"`
	State   string  `json:"state"`
}

type walletClient struct {
	client      *http.Client
	base, token string
}

func (c walletClient) request(path string, body any, csrf string) (int, response, json.RawMessage, error) {
	var data []byte
	var err error
	method := http.MethodGet
	if body != nil {
		method = http.MethodPost
		data, err = json.Marshal(body)
		if err != nil {
			return 0, response{}, nil, err
		}
	}
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(data))
	if err != nil {
		return 0, response{}, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", c.base)
	if csrf != "" {
		req.Header.Set("X-Wallet-CSRF", csrf)
	}
	if c.token != "" {
		req.SetBasicAuth("multi-chain-wallet", c.token)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return 0, response{}, nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, response{}, nil, err
	}
	var value response
	err = json.Unmarshal(raw, &value)
	return res.StatusCode, value, raw, err
}

func run(args []string, token string, client *http.Client, input io.Reader, output io.Writer, password func() ([]byte, error), pause func(time.Duration)) error {
	flags := flag.NewFlagSet("send-and-verify", flag.ContinueOnError)
	flags.SetOutput(output)
	base := flags.String("base-url", "http://localhost:8090", "Local HTTP wallet URL")
	flags.StringVar(&token, "token", token, "Wallet access token (defaults to WALLET_ACCESS_TOKEN)")
	quoteOnly := flags.Bool("test-quote", false, "Check quote without broadcasting")
	send := flags.Bool("send", false, "Confirm and broadcast a Sepolia self-transfer")
	hash := flags.String("hash", "", "Look up an existing transaction")
	if err := flags.Parse(args); err != nil {
		return err
	}
	modes := 0
	if *quoteOnly {
		modes++
	}
	if *send {
		modes++
	}
	if *hash != "" {
		modes++
	}
	if modes != 1 || flags.NArg() != 0 {
		return errors.New("choose exactly one of --test-quote, --send, --hash")
	}
	*base = strings.TrimRight(*base, "/")
	u, err := url.Parse(*base)
	if err != nil || u.Scheme != "http" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return errors.New("only a local HTTP wallet is supported")
	}
	// Never forward credentials or signing requests through a redirect.
	localClient := *client
	localClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	c := walletClient{&localClient, *base, token}
	if *hash != "" {
		if !regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`).MatchString(*hash) {
			return errors.New("invalid transaction hash")
		}
	} else {
		status, network, _, err := c.request("/api/network", nil, "")
		if err != nil {
			return err
		}
		if status != 200 || network.ChainID != 11155111 {
			return errors.New("Sepolia verification failed")
		}
		status, wallet, _, err := c.request("/api/wallet", nil, "")
		if err != nil {
			return err
		}
		if status != 200 || !wallet.Exists {
			return errors.New("create a wallet in the UI first")
		}
		status, balance, _, err := c.request("/api/balance?address="+url.QueryEscape(wallet.Address), nil, "")
		if err != nil {
			return err
		}
		if status != 200 || balance.Wei == nil {
			return errors.New("balance unavailable")
		}
		status, quote, _, err := c.request("/api/wallet/quote", struct {
			Action string `json:"action"`
			To     string `json:"to"`
			Amount string `json:"amount"`
		}{"eth", wallet.Address, "0.000001"}, wallet.CSRF)
		if err != nil {
			return err
		}
		if *quoteOnly {
			if *balance.Wei == "0" {
				if status != 400 || quote.Error != "餘額不足以支付轉帳金額與最高 Gas 手續費" {
					return errors.New("zero-balance guard did not return the expected result")
				}
				fmt.Fprintln(output, "PASS: zero-balance guard; no broadcast")
				return nil
			}
			if status != 200 || quote.ID == "" {
				return fmt.Errorf("quote failed: %s (%d)", quote.Error, status)
			}
			fmt.Fprintln(output, "PASS: quote received; no broadcast")
			return nil
		}
		if status != 200 || quote.ID == "" {
			return fmt.Errorf("quote failed: %s (%d)", quote.Error, status)
		}
		fmt.Fprintf(output, "Sepolia self-transfer: %s %s ETH\nMaximum total: %s ETH\nType SEND to sign and broadcast: ", quote.To, quote.Amount, quote.Total)
		// Read one line without buffering the following terminal password input.
		var line strings.Builder
		b := make([]byte, 1)
		for {
			n, e := input.Read(b)
			if n > 0 {
				if b[0] == '\n' {
					break
				}
				line.WriteByte(b[0])
			}
			if e != nil {
				if e != io.EOF {
					return e
				}
				break
			}
		}
		if strings.TrimSuffix(line.String(), "\r") != "SEND" {
			return errors.New("cancelled")
		}
		fmt.Fprint(output, "Wallet password: ")
		secret, err := password()
		fmt.Fprintln(output)
		if err != nil {
			return err
		}
		status, result, _, err := c.request("/api/wallet/send", struct {
			QuoteID  string `json:"quoteId"`
			Password string `json:"password"`
		}{quote.ID, string(secret)}, wallet.CSRF)
		clear(secret)
		if err != nil {
			return fmt.Errorf("send result unavailable; check history before retrying: %w", err)
		}
		if status != 200 || result.Hash == "" {
			return errors.New("send result unavailable; check history before retrying")
		}
		*hash = result.Hash
		fmt.Fprintf(output, "Recorded: %s state: %s\n", *hash, result.State)
	}
	fmt.Fprintln(output, "https://sepolia.etherscan.io/tx/"+*hash)
	for range 40 {
		status, receipt, raw, err := c.request("/api/transactions/"+*hash, nil, "")
		if err != nil {
			return err
		}
		if status == 200 && (receipt.State == "succeeded" || receipt.State == "reverted") {
			fmt.Fprintln(output, string(raw))
			if receipt.State != "succeeded" {
				return errors.New("execution reverted")
			}
			return nil
		}
		pause(3 * time.Second)
	}
	return errors.New("receipt still unknown; no additional transaction was sent")
}
func main() {
	err := run(os.Args[1:], os.Getenv("WALLET_ACCESS_TOKEN"), &http.Client{Timeout: 60 * time.Second}, os.Stdin, os.Stdout, func() ([]byte, error) { return term.ReadPassword(int(os.Stdin.Fd())) }, time.Sleep)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
