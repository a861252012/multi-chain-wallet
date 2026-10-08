# Transaction recovery verification

## Recovery evidence matrix

These are distinct failure boundaries. Run recovery checks only with disposable wallets and local mock RPCs.

| Scenario | Expected behavior and evidence | Existing verification / limit |
|---|---|---|
| Repeated sends using the same quote | Return the original hash; no second signing/broadcast for that quote | `TestConcurrentSendSignsExactQuoteOnce`: four concurrent callers, one mock broadcast |
| RPC receives bytes and returns an ambiguous error | Persist `broadcast_unknown`; reopen the service and retry the exact bytes/hash | `TestUnknownBroadcastRestartReusesRaw`: mock RPC and service reopen, not an OS process kill or an actual chain receipt |
| A newer journal update arrives while broadcast is delayed | CAS retains the newer record and returns its state, including a reorg state | `TestSendConcurrentStateUpdatePreservesLatest*` and `TestRetryConcurrentStateUpdatePreservesLatest*`: channel-controlled mock broadcast with direct journal updates, not full History/receipt integration |
| Process stops after durable append but before broadcast-state update | A `pending` record may remain; RPC acceptance is unknown from that state alone | `TestProcessKillRestartReusesRaw`: SIGKILL after the local mock receives bytes, before its response; a new Service opens the released process lock, preserves nonce protection and retries identical bytes. This does not cover a kill before the request reaches the mock or power loss. |
| Receipt no longer matches the canonical block | Preserve uncertainty/reorg state and nonce protection | `TestHistoryUpdatesReorgDetectedFromRPC`: mock receipt/header mismatch, not a public-chain reorg |

To rerun the recovery checks, run from the repository root:

```sh
docker run --rm --network none \
  -v "$PWD:/app:ro" \
  -v multi-chain-wallet_go_modules:/go/pkg/mod:ro \
  -e GOCACHE=/tmp/review-cache \
  multi-chain-wallet-app go test -race -count=1 -v ./internal/wallet \
  -run '^(TestProcessKillRestartReusesRaw|TestConcurrentSendSignsExactQuoteOnce|TestUnknownBroadcastRestartReusesRaw|TestSendConcurrentStateUpdatePreservesLatest.*|TestRetryConcurrentStateUpdatePreservesLatest.*|TestHistoryUpdatesReorgDetectedFromRPC)$'
```

This container has no external network and no live wallet volume. Tests use temporary wallet directories and mock RPCs. A passing result covers the assertions above, not power-loss durability or blockchain execution.

The separate [`TestE2EEscrowPaymentReleaseRefundRecovery`](../tests/e2e/escrow_test.go) uses a proxy to drop the RPC response after a simulated EVM accepts the transaction. It reopens the service and journal, retries the same quote without another broadcast, then checks the receipt and order state. Run it with `go test -race -count=1 -run '^TestE2EEscrowPaymentReleaseRefundRecovery$' ./tests/e2e`. This covers local integration with a simulated EVM, not response loss on public Sepolia.
