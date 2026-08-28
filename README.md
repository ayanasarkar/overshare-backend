# Overshare — Backend (Go)

Backend service for Overshare: image scan (EXIF GPS + OCR/object/document
flags), fix (strip metadata / blur regions), and certify (IPFS + local
blockchain proof-of-fix). Built in Go with Gin.

Part of a two-track project — see [Wrik's AI/ML + Frontend repo](https://github.com/Wriksg/OVERSHARE)
for the counterpart service this talks to.

## Status

| Component | Status |
|---|---|
| `/api/scan` (Gin routes, cache-first lookup) | ✅ Working |
| EXIF GPS extraction (`metadata/exif.go`) | ⚠️ Built, verified only via precomputed cache — live (cache-miss) path not yet tested |
| Precomputed demo cache (`demo-assets/precomputed_scans.json`) | ⚠️ 1 of 5 curated images done |
| `/api/fix` (strip metadata, blur regions) | ✅ Working — verified with real output file + visual check |
| `/api/certify` (IPFS + local chain mint) | ✅ Working end-to-end — real mint tested (cert_id, tx_hash, IPFS CID all returned) |
| Fix 10 gate (certify requires a fixed image first) | ✅ Verified, including against a re-scan edge case (see below) |
| Live AI-service integration (`/ocr`, `/detect`, `/document-flags`, `/explain`) | ⛔ Not wired up yet — cache-miss scan calls fail cleanly with a clear error until Wrik's `ai_service` is live on `:8001` |

## Architecture

```
[Wrik's ai_service — FastAPI, :8001]
        |
        v  (only on a cache miss — judged demo path never reaches this)
[this service — Go/Gin, :8080]  --(Kubo, local IPFS)--> IPFS
                                 --(Anvil, local chain)--> OvershareCertificate.sol
        |
        v  REST: /api/scan, /api/fix, /api/certify
[Wrik's React frontend]
```

## Running locally

Requires: Go, Tesseract (for the live/cache-miss OCR path), and — only if
you want certify working — [Foundry](https://getfoundry.sh) (Anvil) and
[Kubo](https://docs.ipfs.tech/install/command-line/) (IPFS).

```bash
go run .
```

Core scan/fix path works with nothing else running. Certify is opt-in —
without the env vars below, the server logs `certify feature disabled`
and scan/fix are unaffected (verified: the core path works with those
services entirely stopped).

### Enabling certify

1. Start Anvil: `anvil` (separate terminal, leave running)
2. Deploy the contract:
   ```bash
   cd contract-deploy
   forge build
   forge create src/OvershareCertificate.sol:OvershareCertificate \
     --rpc-url http://127.0.0.1:8545 \
     --private-key <anvil-account-0-private-key> \
     --broadcast
   ```
3. Start Kubo: `ipfs daemon` (separate terminal, leave running).
   Note: Kubo's Gateway defaults to port 8080, which collides with this
   server. Point it elsewhere first:
   ```bash
   ipfs config Addresses.Gateway /ip4/127.0.0.1/tcp/8081
   ```
4. Set env vars and start the Go server:
   ```powershell
   $env:CERTIFICATE_CONTRACT_ADDRESS="<deployed address>"
   $env:ANVIL_RPC_URL="http://localhost:8545"
   $env:CERTIFIER_PRIVATE_KEY="<anvil account 0 private key, no 0x prefix>"
   go run .
   ```
   Confirm the startup log shows `certify feature enabled (IPFS + Anvil connected)`.

## Known limitations

- **In-memory store** (`store/store.go`) does not survive a server
  restart — by design for demo scope, but every scan/fix record is lost
  on restart. Re-run scan → fix before testing certify after any restart.
- **Precomputed cache is incomplete** — only 1 of the 5 planned curated
  demo images has real data in `demo-assets/precomputed_scans.json`.

## Bug fixed during testing

Re-scanning an already-fixed image used to silently wipe its
`FixedImagePath`, because the cache-hit branch in `scan.go` created a
fresh store record on every call instead of preserving existing fix
state. Fixed by checking for an existing record and carrying its
`FixedImagePath` forward before overwriting. Verified by reproducing the
failure, applying the fix, and re-running the same sequence.

## Repo layout

```
handlers/     — /api/scan, /api/fix, /api/certify
metadata/     — EXIF GPS extraction
imaging/      — strip-metadata / blur-region fix actions
clients/      — IPFS (Kubo) and chain (Anvil/go-ethereum) clients
contracts/    — hand-written Go binding for OvershareCertificate.sol
models/       — shared request/response schemas (contract with frontend)
store/        — in-memory scan/fix record store
demo-assets/  — precomputed_scans.json (curated demo cache)
contract-deploy/ — standalone Foundry project for deploying the contract
```
