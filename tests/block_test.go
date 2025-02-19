package tests

import (
	"time"

	"github.com/pactus-project/pactus/crypto/hash"
	"github.com/pactus-project/pactus/util/logger"
	pactus "github.com/pactus-project/pactus/www/grpc/gen/go"
)

func lastHash() hash.Hash {
	res, err := tBlockchain.GetBlockchainInfo(tCtx,
		&pactus.GetBlockchainInfoRequest{})
	if err != nil {
		panic(err)
	}
	h, _ := hash.FromBytes(res.LastBlockHash)
	return h
}

func lastHeight() uint32 {
	res, err := tBlockchain.GetBlockchainInfo(tCtx,
		&pactus.GetBlockchainInfoRequest{})
	if err != nil {
		panic(err)
	}
	return res.LastBlockHeight
}

func waitForNewBlocks(num uint32) {
	height := lastHeight() + num
	for i := uint32(0); i < num; i++ {
		if lastHeight() > height {
			break
		}
		time.Sleep(2 * time.Second)
	}
}

func lastBlock() *pactus.GetBlockResponse {
	return getBlockAt(lastHeight())
}

func getBlockAt(height uint32) *pactus.GetBlockResponse {
	for i := 0; i < 120; i++ {
		res, err := tBlockchain.GetBlock(tCtx,
			&pactus.GetBlockRequest{
				Height:    height,
				Verbosity: pactus.BlockVerbosity_BLOCK_INFO,
			},
		)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		return res
	}
	logger.Panic("get block timeout", "height", height)
	return nil
}
