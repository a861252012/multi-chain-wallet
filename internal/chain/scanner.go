package chain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func (c *Client) FinalizedNumber(ctx context.Context) (uint64, error) {
	h, err := c.HeaderByNumber(ctx, big.NewInt(-3))
	if err != nil {
		return 0, err
	}
	return h.Number.Uint64(), nil
}

// ScanFinalizedBlock discovers candidates from block bodies and all receipts, independent of token filters.
// Amounts are still reconstructed and verified by Activity, never trusted from discovery alone.
func (c *Client) ScanFinalizedBlock(ctx context.Context, number uint64, owner common.Address) (*ScannedBlock, error) {
	final, err := c.FinalizedNumber(ctx)
	if err != nil {
		return nil, err
	}
	if number > final {
		return nil, ErrUnavailable
	}
	var block rpcBlock
	if err := c.rpc.Client().CallContext(ctx, &block, "eth_getBlockByNumber", hexutil.EncodeUint64(number), true); err != nil {
		return nil, ErrUnavailable
	}
	domainBlock, err := rpcBlockToDomain(block)
	if err != nil || domainBlock.hash == (common.Hash{}) {
		return nil, ErrUnavailable
	}
	var receipts []*types.Receipt
	if err := c.rpc.Client().CallContext(ctx, &receipts, "eth_getBlockReceipts", hexutil.EncodeUint64(number)); err != nil {
		for _, tx := range domainBlock.transactions {
			r, err := c.rpc.TransactionReceipt(ctx, tx.hash)
			if err != nil {
				return nil, rpcError(err)
			}
			receipts = append(receipts, r)
		}
	}
	if len(receipts) != len(domainBlock.transactions) {
		return nil, ErrUnavailable
	}
	result := scannedBlockRecord{hash: domainBlock.hash, hashes: []common.Hash{}, tokens: []common.Address{}}
	hashes, tokens := map[common.Hash]bool{}, map[common.Address]bool{}
	ownerTopic := common.BytesToHash(owner.Bytes())
	for i, tx := range domainBlock.transactions {
		r := receipts[i]
		if r == nil || r.BlockHash != domainBlock.hash || r.TxHash != tx.hash || r.BlockNumber == nil || r.BlockNumber.Uint64() != number {
			return nil, ErrUnavailable
		}
		related := tx.from == owner || (tx.to != nil && *tx.to == owner)
		if r.Status == types.ReceiptStatusSuccessful {
			if _, err := nativeTransferLogs(r, c.ChainID()); err != nil {
				return nil, err
			}
			for _, l := range r.Logs {
				if len(l.Topics) != 3 || l.Topics[0] != transferTopic || len(l.Data) != 32 || (l.Topics[1] != ownerTopic && l.Topics[2] != ownerTopic) {
					continue
				}
				related = true
				if IsNativeTransferEmitter(c.ChainID(), l.Address) {
					continue
				}
				if !tokens[l.Address] {
					result.tokens = append(result.tokens, l.Address)
					tokens[l.Address] = true
				}
			}
		}
		if related && !hashes[tx.hash] {
			result.hashes = append(result.hashes, tx.hash)
			hashes[tx.hash] = true
		}
	}
	h, err := c.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil || h.Hash() != domainBlock.hash {
		return nil, ErrUnavailable
	}
	return scannedBlockToStorage(result), nil
}
