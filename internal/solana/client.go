package solana

import (
	"context"
	"fmt"
	"strings"
	"time"

	solanaSDK "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type Config struct {
	RPCURL string
	ProgramID string
}

type Client struct {
	rpc *rpc.Client
	programID solanaSDK.PublicKey
}

func NewClient(cfg Config)(*Client, error) {
	rpcURL := strings.TrimSpace(cfg.RPCURL)
	programID := strings.TrimSpace(cfg.ProgramID)

	if rpcURL == "" {
		return nil, fmt.Errorf("SOLANA_RPC_URL is required")
	}

	if programID == "" {
		return nil, fmt.Errorf("SOLANA_PROGRAM_ID is required")
	}

	// Validate the program ID.
	pubKey, err := solanaSDK.PublicKeyFromBase58(programID)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana program ID: %w", err)
	}

	return &Client{
		rpc: rpc.New(rpcURL),
		programID: pubKey,

	}, nil
}

// HealthCheck verifies connectivity with solana RPC node.
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.rpc.GetSlot(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("Solana RPC health check failed: %w", err)
	}

	return nil
}

// ProgramID returns the configured escrow program ID.
func (c *Client) ProgramID() solanaSDK.PublicKey {
	return c.programID
}