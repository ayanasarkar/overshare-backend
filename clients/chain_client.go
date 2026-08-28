package clients

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	// Hand-written binding built directly from the ABI (no abigen step) —
	// see contracts/certificate.go's header comment for why, and for how
	// to swap in real abigen output instead if you'd rather.
	"overshare-backend/contracts"
)

// ChainClient wraps go-ethereum's ethclient + the abigen-generated contract
// binding for OvershareCertificate, targeting a local Anvil node.
type ChainClient struct {
	ethClient       *ethclient.Client
	contract        *contracts.OvershareCertificate
	auth            *bind.TransactOpts
	contractAddress common.Address
}

// NewChainClient connects to the local Anvil RPC, binds the deployed
// contract at contractAddrHex, and sets up a transactor from one of Anvil's
// pre-funded test private keys.
func NewChainClient(rpcURL string, contractAddrHex string, privateKeyHex string) (*ChainClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial chain node at %s: %w", rpcURL, err)
	}

	contractAddr := common.HexToAddress(contractAddrHex)
	cert, err := contracts.NewOvershareCertificate(contractAddr, client)
	if err != nil {
		return nil, fmt.Errorf("bind OvershareCertificate contract: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get chain id from %s: %w", rpcURL, err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("create transactor: %w", err)
	}

	return &ChainClient{
		ethClient:       client,
		contract:        cert,
		auth:            auth,
		contractAddress: contractAddr,
	}, nil
}

// MintCertificate calls mintCertificate(imageHash, ipfsCID), waits for the
// transaction to be mined, and reads the on-chain cert ID back out of the
// emitted CertificateMinted event (rather than guessing it from nextId,
// which could race under concurrent calls).
func (c *ChainClient) MintCertificate(imageHash [32]byte, ipfsCID string) (certID uint64, txHash string, err error) {
	tx, err := c.contract.MintCertificate(c.auth, imageHash, ipfsCID)
	if err != nil {
		return 0, "", fmt.Errorf("send mintCertificate tx: %w", err)
	}

	receipt, err := bind.WaitMined(context.Background(), c.ethClient, tx)
	if err != nil {
		return 0, "", fmt.Errorf("wait for receipt: %w", err)
	}

	for _, vLog := range receipt.Logs {
		event, parseErr := c.contract.ParseCertificateMinted(*vLog)
		if parseErr == nil {
			return event.Id.Uint64(), tx.Hash().Hex(), nil
		}
	}

	return 0, tx.Hash().Hex(), fmt.Errorf("tx mined but no CertificateMinted event found in receipt")
}
