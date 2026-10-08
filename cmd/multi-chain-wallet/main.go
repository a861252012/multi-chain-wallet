package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/a861252012/multi-chain-wallet/internal/chain"
	"github.com/a861252012/multi-chain-wallet/internal/wallet"
	"github.com/a861252012/multi-chain-wallet/internal/web"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--version":
			fmt.Println(version)
			return nil
		case "--healthcheck":
			port := os.Getenv("PORT")
			if port == "" {
				port = strconv.Itoa(defaultHTTPPort)
			}
			client := &http.Client{Timeout: 3 * time.Second}
			response, err := client.Get("http://127.0.0.1:" + port + "/healthz")
			if err != nil {
				return errors.New("health check connection failed")
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return errors.New("health check failed")
			}
			return nil
		}
	}
	config, err := loadConfig(os.Getenv)
	if err != nil {
		return err
	}
	client, err := chain.NewFallback(config.sepoliaRPCURLs)
	if err != nil {
		return err
	}
	defer client.Close()
	walletService, err := wallet.NewService(client, config.walletDir)
	if err != nil {
		return err
	}
	defer walletService.Close()
	if config.sepoliaVault != "" {
		if err := walletService.SetVaultAddress(config.sepoliaVault); err != nil {
			return err
		}
	}
	if config.sepoliaEscrow != "" {
		if err := walletService.SetEscrow(config.sepoliaEscrow, wallet.USDCAddress); err != nil {
			return err
		}
	}
	if config.sharedDemo {
		info, err := walletService.Status()
		if err != nil {
			return err
		}
		if !info.Exists {
			return errors.New("共用 Demo 必須先建立測試錢包")
		}
	}
	handler, err := web.New(client, walletService)
	if err != nil {
		return err
	}
	extraClients := map[string]*chain.Client{}
	for _, network := range config.networks {
		c, err := chain.NewNetwork(network.chainID, network.rpcURLs)
		if err != nil {
			return err
		}
		defer c.Close()
		extraClients[network.slug] = c
	}
	// Each network owns its journal and quote store; linked services share only the account keystore.
	buildNetworks := func(primary *wallet.Service, dir string, first http.Handler) (http.Handler, []*wallet.Service, error) {
		mux := http.NewServeMux()
		mux.Handle("/", first)
		var services []*wallet.Service
		for slug, c := range extraClients {
			service, err := wallet.NewLinkedService(c, filepath.Join(dir, "networks", strconv.FormatInt(c.ChainID(), 10)), primary)
			if err != nil {
				for _, s := range services {
					s.Close()
				}
				return nil, nil, err
			}
			services = append(services, service)
			h, err := web.New(c, service)
			if err != nil {
				for _, s := range services {
					s.Close()
				}
				return nil, nil, err
			}
			prefix := "/net/" + slug
			mux.Handle(prefix+"/", http.StripPrefix(prefix, h))
		}
		return mux, services, nil
	}
	networks, linked, err := buildNetworks(walletService, config.walletDir, handler)
	if err != nil {
		return err
	}
	defer func() {
		for _, service := range linked {
			service.Close()
		}
	}()
	maintenanceCtx, stopMaintenance := context.WithCancel(context.Background())
	var maintenance sync.WaitGroup
	startMaintenance := func(service *wallet.Service) {
		maintenance.Go(func() {
			service.RunMaintenance(maintenanceCtx)
		})
	}
	startMaintenance(walletService)
	for _, service := range linked {
		startMaintenance(service)
	}
	var accountsMu sync.Mutex
	type accountWorkspace struct {
		handler  http.Handler
		services []*wallet.Service
	}
	accountWorkspaces := map[string]accountWorkspace{}
	var accountServices []*wallet.Service
	shuttingDown := false
	errShuttingDown := errors.New("服務正在停止")
	defer func() {
		accountsMu.Lock()
		shuttingDown = true
		services := append([]*wallet.Service{}, accountServices...)
		accountsMu.Unlock()
		stopMaintenance()
		maintenance.Wait()
		for _, service := range services {
			service.Close()
		}
	}()
	// Startup, faucet access and HTTP requests share one service per account.
	loadAccount := func(id string) (accountWorkspace, error) {
		accountsMu.Lock()
		defer accountsMu.Unlock()
		if shuttingDown {
			return accountWorkspace{}, errShuttingDown
		}
		if workspace, ok := accountWorkspaces[id]; ok {
			return workspace, nil
		}
		primary, err := wallet.NewAccountService(client, walletService, id)
		if err != nil {
			return accountWorkspace{}, err
		}
		first, err := web.New(client, primary)
		if err != nil {
			primary.Close()
			return accountWorkspace{}, err
		}
		mux, children, err := buildNetworks(primary, filepath.Join(config.walletDir, "accounts", id), first)
		if err != nil {
			primary.Close()
			return accountWorkspace{}, err
		}
		workspace := accountWorkspace{
			handler:  http.StripPrefix("/accounts/"+id, mux),
			services: append([]*wallet.Service{primary}, children...),
		}
		accountWorkspaces[id] = workspace
		accountServices = append(accountServices, workspace.services...)
		for _, service := range workspace.services {
			startMaintenance(service)
		}
		return workspace, nil
	}
	accounts, err := walletService.Accounts()
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if account.ID == "" {
			continue
		}
		// Archiving only hides the account; its pending transactions still need tracking.
		if _, err := loadAccount(account.ID); err != nil {
			return fmt.Errorf("載入帳戶 %s: %w", account.ID, err)
		}
	}

	solWallet, err := wallet.NewSolanaService(config.solanaRPC, filepath.Join(config.walletDir, "solana-devnet"), 0)
	if err != nil {
		return err
	}
	defer solWallet.Close()
	solHandler, err := web.NewSolana(solWallet, walletService.CSRFToken())
	if err != nil {
		return err
	}
	tronWallet, err := wallet.NewTronService(config.tronRPC, config.tronAPIKey, filepath.Join(config.walletDir, "tron-shasta"), 0)
	if err != nil {
		return err
	}
	defer tronWallet.Close()
	tronHandler, err := web.NewTron(tronWallet, walletService.CSRFToken())
	if err != nil {
		return err
	}
	faucet := &wallet.TestFaucet{Root: walletService, Sources: map[int64]*wallet.Service{}}
	if config.faucetPath != "" {
		faucetConfig, err := readFaucetConfig(config.faucetPath)
		if err != nil {
			return err
		}
		workspace, err := loadAccount(faucetConfig.AccountID)
		if err != nil {
			return err
		}
		if faucetConfig.TronDir != "" && faucetConfig.TronPassword != "" {
			source, err := wallet.NewTronService(config.tronRPC, config.tronAPIKey, faucetConfig.TronDir, 0)
			if err != nil {
				return err
			}
			defer source.Close()
			faucet.TronSource, faucet.TronRecipient, faucet.TronPassword = source, tronWallet, faucetConfig.TronPassword
		}
		faucet.Password = faucetConfig.Password
		for _, service := range workspace.services {
			status, err := service.Status()
			if err != nil {
				return err
			}
			faucet.Sources[status.ChainID] = service
		}
	}
	faucetHandler := web.NewFaucet(faucet, solWallet, walletService.CSRFToken())
	workspace := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.Header().Set("X-App-Version", version)
		}
		if r.URL.Path == "/api/faucet" || strings.HasPrefix(r.URL.Path, "/api/faucet/") {
			faucetHandler.ServeHTTP(w, r)
			return
		}
		if target, ok := web.NormalizeWorkspaceRedirect(r.URL.Path); ok {
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/tron/") {
			http.StripPrefix("/tron", tronHandler).ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/solana/") {
			http.StripPrefix("/solana", solHandler).ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/accounts/") {
			networks.ServeHTTP(w, r)
			return
		}
		pieces := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/accounts/"), "/", 2)
		if len(pieces) != 2 || pieces[0] == "" {
			http.NotFound(w, r)
			return
		}
		account, err := loadAccount(pieces[0])
		if err != nil {
			if errors.Is(err, errShuttingDown) {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			http.NotFound(w, r)
			return
		}
		account.handler.ServeHTTP(w, r)
	})
	var accessHandler http.Handler = web.RequireAccessToken(workspace, config.accessToken)
	if config.sharedDemo {
		accessHandler = web.SharedDemo(workspace, config.accessToken)
	}
	server := &http.Server{
		Addr: net.JoinHostPort(config.httpHost, strconv.Itoa(config.httpPort)), Handler: web.WithPublicOrigin(accessHandler, config.publicOrigin),
		MaxHeaderBytes:    32 * 1024,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverError := make(chan error, 1)
	go func() { serverError <- server.ListenAndServe() }()
	slog.Info("Multi-Chain Wallet listening", "url", "http://"+server.Addr, "mode", "Sepolia wallet")
	select {
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.New("HTTP 服務啟動失敗；請檢查 PORT 是否已被使用")
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}
