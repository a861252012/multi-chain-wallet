package main

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigAppliesDefaultsAtTheProcessBoundary(t *testing.T) {
	getenv := func(key string) string {
		if key == "WALLET_ACCESS_TOKEN" {
			return strings.Repeat("a", 32)
		}
		return ""
	}

	config, err := loadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if config.sepoliaVault != "" {
		t.Fatalf("sepoliaVault default: got %q, want empty", config.sepoliaVault)
	}
	if config.httpHost != defaultHTTPHost || config.httpPort != defaultHTTPPort {
		t.Fatalf("http defaults: %#v", config)
	}
	if config.walletDir != defaultWalletDir || config.solanaRPC != defaultSolanaRPC || config.tronRPC != defaultTronRPC {
		t.Fatalf("service defaults: %#v", config)
	}
	if len(config.sepoliaRPCURLs) != 1 || config.sepoliaRPCURLs[0] != defaultSepoliaRPC {
		t.Fatalf("sepolia endpoints: %#v", config.sepoliaRPCURLs)
	}
	if len(config.networks) != len(networkDefinitions) {
		t.Fatalf("network count: got %d want %d", len(config.networks), len(networkDefinitions))
	}
	for index, network := range config.networks {
		definition := networkDefinitions[index]
		if network.slug != definition.slug || network.chainID != definition.chainID || len(network.rpcURLs) != 1 || network.rpcURLs[0] != definition.defaultRPC {
			t.Fatalf("network %d: %#v", index, network)
		}
	}
}

func TestLoadConfigValidatesProcessInputs(t *testing.T) {
	token := strings.Repeat("b", 32)
	base := map[string]string{"WALLET_ACCESS_TOKEN": token}
	for _, tc := range []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{name: "port", key: "PORT", value: "0", want: "PORT 必須是 1–65535 的整數"},
		{name: "port text", key: "PORT", value: "nope", want: "PORT 必須是 1–65535 的整數"},
		{name: "host", key: "HTTP_HOST", value: "0.0.0.1", want: "HTTP_HOST 僅允許 127.0.0.1 或 0.0.0.0"},
		{name: "token", key: "WALLET_ACCESS_TOKEN", value: "short", want: "若設定 WALLET_ACCESS_TOKEN，必須至少 32 字元"},
		{name: "vault invalid", key: "SEPOLIA_VAULT_ADDRESS", value: "0x123", want: "SEPOLIA_VAULT_ADDRESS 格式錯誤"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := make(map[string]string, len(base)+1)
			maps.Copy(values, base)
			values[tc.key] = tc.value
			_, err := loadConfig(func(key string) string { return values[key] })
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestLoadConfigTrimsFallbackEndpoints(t *testing.T) {
	values := map[string]string{
		"WALLET_ACCESS_TOKEN":                strings.Repeat("c", 32),
		"SEPOLIA_RPC_URL":                    "https://primary.example/rpc",
		"SEPOLIA_RPC_FALLBACK_URLS":          " https://one.example/rpc,https://two.example/rpc ",
		"ARBITRUM_SEPOLIA_RPC_URL":           "https://arbitrum.example/rpc",
		"ARBITRUM_SEPOLIA_RPC_FALLBACK_URLS": "https://arbitrum-backup.example/rpc",
	}
	config, err := loadConfig(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(config.sepoliaRPCURLs, ","), "https://primary.example/rpc,https://one.example/rpc,https://two.example/rpc"; got != want {
		t.Fatalf("sepolia endpoints = %q, want %q", got, want)
	}
	if got, want := strings.Join(config.networks[0].rpcURLs, ","), "https://arbitrum.example/rpc,https://arbitrum-backup.example/rpc"; got != want {
		t.Fatalf("arbitrum endpoints = %q, want %q", got, want)
	}
}

func TestReadFaucetConfigValidatesPrivateFileBoundary(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "faucet.json")
	if err := os.WriteFile(filename, []byte(`{"accountId":"fixture","password":"secret","tronDir":"/tmp/tron","tronPassword":"tron-secret"}`), 0600); err != nil {
		t.Fatal(err)
	}
	config, err := readFaucetConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	if config.AccountID != "fixture" || config.Password != "secret" || config.TronDir != "/tmp/tron" || config.TronPassword != "tron-secret" {
		t.Fatalf("config = %#v", config)
	}

	if err := os.Chmod(filename, 0644); err != nil {
		t.Fatal(err)
	}
	_, err = readFaucetConfig(filename)
	if err == nil || err.Error() != "TEST_FAUCET_CONFIG 必須是僅擁有者可讀寫的設定檔（0600）" {
		t.Fatalf("insecure mode error = %v", err)
	}

	missing := filepath.Join(dir, "missing.json")
	_, err = readFaucetConfig(missing)
	if err == nil || err.Error() != "TEST_FAUCET_CONFIG 必須是僅擁有者可讀寫的設定檔（0600）" {
		t.Fatalf("missing error = %v", err)
	}

	if err := os.Chmod(filename, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(`{"accountId":""}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = readFaucetConfig(filename)
	if err == nil || err.Error() != "測試幣發放設定缺少專用 accountId 或 password" {
		t.Fatalf("invalid content error = %v", err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatal("content error leaked filesystem detail")
	}
}

func TestPublicDeploymentConfig(t *testing.T) {
	for _, origin := range []string{"https://wallet.example", "https://wallet.example:8443"} {
		c, err := loadConfig(func(key string) string {
			return map[string]string{"PUBLIC_ORIGIN": origin, "WALLET_ACCESS_TOKEN": strings.Repeat("a", 32)}[key]
		})
		if err != nil || c.publicOrigin != origin {
			t.Fatalf("valid origin %q: %v", origin, err)
		}
	}
	for _, origin := range []string{"http://wallet.example", "https://", "https://user:pass@wallet.example", "https://wallet.example/", "https://wallet.example?", "https://wallet.example#", "https://wallet.example/path", "https://wallet.example?x=y", "https://wallet.example#x"} {
		_, err := loadConfig(func(key string) string {
			return map[string]string{"PUBLIC_ORIGIN": origin, "WALLET_ACCESS_TOKEN": strings.Repeat("a", 32)}[key]
		})
		if err == nil {
			t.Fatalf("unsafe origin accepted: %q", origin)
		}
	}
	_, err := loadConfig(func(key string) string {
		if key == "PUBLIC_ORIGIN" {
			return "https://wallet.example"
		}
		return ""
	})
	if err == nil {
		t.Fatal("public deployment without token accepted")
	}
}

func TestSharedDemoRequiresExplicitHTTPSOrigin(t *testing.T) {
	for _, values := range []map[string]string{
		{"SHARED_DEMO": "true"}, {"SHARED_DEMO": "yes", "PUBLIC_ORIGIN": "https://wallet.example"},
	} {
		if _, err := loadConfig(func(key string) string { return values[key] }); err == nil {
			t.Fatal("unsafe shared configuration accepted")
		}
	}
	config, err := loadConfig(func(key string) string {
		return map[string]string{"SHARED_DEMO": "true", "PUBLIC_ORIGIN": "https://wallet.example"}[key]
	})
	if err != nil || !config.sharedDemo {
		t.Fatalf("explicit shared mode rejected: %v", err)
	}
}

func TestEscrowAddressConfig(t *testing.T) {
	for _, value := range []string{"", "0x1111111111111111111111111111111111111111", "0x0000000000000000000000000000000000000000", "not-an-address"} {
		config, err := loadConfig(func(key string) string {
			if key == "SEPOLIA_ESCROW_ADDRESS" {
				return value
			}
			return ""
		})
		valid := value == "" || value == "0x1111111111111111111111111111111111111111"
		if (err == nil) != valid {
			t.Fatalf("value=%s err=%v", value, err)
		}
		if valid && config.sepoliaEscrow != value {
			t.Fatal("configuration not preserved")
		}
	}
}
