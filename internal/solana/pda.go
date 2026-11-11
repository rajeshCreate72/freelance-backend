package solana

import (
	"encoding/binary"
	"fmt"

	solanaSDK "github.com/gagliardetto/solana-go"
)

// DeriveEscrowPDA derives the escrow PDA.
//
// Seeds:
// 1. "escrow" - byte string
// 2. Client public key bytes
// 3. Job ID encoded as little-endian uint64
func (c *Client) DeriveEscrowPDA(
	client solanaSDK.PublicKey,
	jobID uint64,
) (solanaSDK.PublicKey, uint8, error) {
	// Convert job ID into 8-byte little-endian representation.
	jobIDBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(jobIDBytes, jobID)

	// Match the seeds defined in the Anchor program.
	seeds := [][]byte{
		[]byte("escrow"),
		client.Bytes(),
		jobIDBytes,
	}

	// Derive the PDA using the escrow program ID.
	pda, bump, err := solanaSDK.FindProgramAddress(
		seeds,
		c.programID,
	)
	if err != nil {
		return solanaSDK.PublicKey{}, 0,
			fmt.Errorf("failed to derive escrow PDA: %w", err)
	}

	return pda, bump, nil
}