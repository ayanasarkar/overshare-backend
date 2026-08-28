package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"overshare-backend/clients"
	"overshare-backend/handlers"
)

func main() {
	if err := handlers.LoadPrecomputedCache(); err != nil {
		log.Fatalf("failed to load precomputed cache: %v", err)
	}

	// Certify is opt-in and independent of the core path (No-Lag Checklist:
	// "verify by unplugging wifi and confirming Upload -> Scan -> Fix still
	// works"). Only wire up IPFS/chain if configured — otherwise scan/fix
	// run standalone with nothing else even attempted.
	if contractAddr := os.Getenv("CERTIFICATE_CONTRACT_ADDRESS"); contractAddr != "" {
		ipfsClient := clients.NewIPFSClient(envOr("IPFS_API_URL", "http://localhost:5001"))

		chainClient, err := clients.NewChainClient(
			envOr("ANVIL_RPC_URL", "http://localhost:8545"),
			contractAddr,
			os.Getenv("CERTIFIER_PRIVATE_KEY"),
		)
		if err != nil {
			log.Printf("certify feature disabled — chain client init failed: %v", err)
		} else {
			handlers.InitCertifyClients(ipfsClient, chainClient)
			log.Println("certify feature enabled (IPFS + Anvil connected)")
		}
	} else {
		log.Println("certify feature disabled — CERTIFICATE_CONTRACT_ADDRESS not set (core scan/fix path is unaffected)")
	}

	router := gin.Default()

	api := router.Group("/api")
	{
		api.POST("/scan", handlers.ScanHandler)
		api.POST("/fix", handlers.FixHandler)
		api.POST("/certify", handlers.CertifyHandler)
	}

	log.Println("overshare backend listening on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
