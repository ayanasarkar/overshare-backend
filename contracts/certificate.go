// Package contracts is a hand-written go-ethereum binding for
// OvershareCertificate.sol, built directly from its ABI via
// accounts/abi + accounts/abi/bind. This intentionally skips the abigen
// step mentioned in the doc's B6/review — one less toolchain dependency
// (no need to install abigen or wire forge's output into it) for two people
// to keep working. Functionally it's the same shape abigen would generate
// (NewOvershareCertificate / MintCertificate / ParseCertificateMinted), so
// clients/chain_client.go doesn't need to know which one it's talking to.
//
// If you'd rather use real abigen-generated bindings instead, delete this
// file and run:
//
//	forge build
//	jq '.abi' out/OvershareCertificate.sol/OvershareCertificate.json > contracts/OvershareCertificate.abi
//	abigen --abi=contracts/OvershareCertificate.abi --pkg=contracts --type=OvershareCertificate --out=contracts/certificate.go
package contracts

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// certificateABIJSON must stay in sync with contracts/OvershareCertificate.sol.
const certificateABIJSON = `[
	{
		"type": "function",
		"name": "mintCertificate",
		"stateMutability": "nonpayable",
		"inputs": [
			{"name": "imageHash", "type": "bytes32"},
			{"name": "ipfsCID", "type": "string"}
		],
		"outputs": [
			{"name": "", "type": "uint256"}
		]
	},
	{
		"type": "function",
		"name": "nextId",
		"stateMutability": "view",
		"inputs": [],
		"outputs": [
			{"name": "", "type": "uint256"}
		]
	},
	{
		"type": "function",
		"name": "certificates",
		"stateMutability": "view",
		"inputs": [
			{"name": "", "type": "uint256"}
		],
		"outputs": [
			{"name": "imageHash", "type": "bytes32"},
			{"name": "ipfsCID", "type": "string"},
			{"name": "timestamp", "type": "uint256"}
		]
	},
	{
		"type": "event",
		"name": "CertificateMinted",
		"anonymous": false,
		"inputs": [
			{"name": "id", "type": "uint256", "indexed": true},
			{"name": "imageHash", "type": "bytes32", "indexed": false},
			{"name": "ipfsCID", "type": "string", "indexed": false}
		]
	}
]`

// OvershareCertificate is a thin, typed wrapper around a bind.BoundContract
// pointed at a deployed OvershareCertificate.sol instance.
type OvershareCertificate struct {
	address  common.Address
	contract *bind.BoundContract
}

// NewOvershareCertificate binds to a deployed contract at address, using
// backend for calls/transactions/log filtering (an *ethclient.Client
// satisfies all three roles).
func NewOvershareCertificate(address common.Address, backend bind.ContractBackend) (*OvershareCertificate, error) {
	parsed, err := abi.JSON(strings.NewReader(certificateABIJSON))
	if err != nil {
		return nil, err
	}
	contract := bind.NewBoundContract(address, parsed, backend, backend, backend)
	return &OvershareCertificate{address: address, contract: contract}, nil
}

// MintCertificate sends a mintCertificate(imageHash, ipfsCID) transaction.
func (o *OvershareCertificate) MintCertificate(opts *bind.TransactOpts, imageHash [32]byte, ipfsCID string) (*types.Transaction, error) {
	return o.contract.Transact(opts, "mintCertificate", imageHash, ipfsCID)
}

// OvershareCertificateCertificateMinted mirrors the CertificateMinted event.
type OvershareCertificateCertificateMinted struct {
	Id        *big.Int
	ImageHash [32]byte
	IpfsCID   string
	Raw       types.Log
}

// ParseCertificateMinted decodes a raw log into a CertificateMinted event,
// the same pattern abigen-generated code uses.
func (o *OvershareCertificate) ParseCertificateMinted(log types.Log) (*OvershareCertificateCertificateMinted, error) {
	event := new(OvershareCertificateCertificateMinted)
	if err := o.contract.UnpackLog(event, "CertificateMinted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
