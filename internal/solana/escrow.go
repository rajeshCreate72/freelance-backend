package solana

import (
	"context"
	"encoding/binary"
	"fmt"

	solanaSDK "github.com/gagliardetto/solana-go"
)

var escrowStateDiscriminator = [8]byte{
	19, 90, 148, 111, 55, 130, 229, 108,
}

const (
	EscrowDiscriminator = 8
	EscrowAccountSize = 129
)

// Status represents the on-chain EscrowState status.

// These values follow the rust enum variants
// Borsh serializes these enum variants as a single byte.
type Status uint8

const (
	StatusFunded Status = iota
	StatusAccepted
	StatusReleased
	StatusCompleted
	StatusCancelled
	StatusPending
)

func (s Status) String() string {
	switch s {
	case StatusFunded:
		return "Funded"
	case StatusAccepted:
		return "Accepted"
	case StatusReleased:
		return "Released"
	case StatusCompleted:
		return "Completed"
	case StatusCancelled:
		return "Cancelled"
	case StatusPending:
		return "Pending"
	default:
		return "Unknown"
	}
}

// EscrowState represents the decoded on-chain escrow account.
type EscrowState struct {
	Client solanaSDK.PublicKey
	Freelancer solanaSDK.PublicKey
	Amount uint64
	Status Status
	Deadline int64
	GracePeriod int64
	ReviewWindow int64
	CompletedAt int64
	JobID uint64
}

// GetEscrow fetches an escrow account from Solana and decodes its data.
func (c *Client) GetEscrow(
	ctx context.Context,
	escrowAddress solanaSDK.PublicKey,
) (*EscrowState, error) {
	
	// Fetch account information from Solana RPC
	result, err := c.rpc.GetAccountInfo(ctx, escrowAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch escrow account: %w", err)
	}

	// Account doesn't exisit
	if result == nil || result.Value == nil {
		return nil, fmt.Errorf("escrow account not found: %s", escrowAddress)
	}

	// Verify that account is owned by our escrow program.
	if result.Value.Owner != c.programID {
		return nil, fmt.Errorf(
			"invalid escrow account owner: expected %s, got %s",
			c.programID,
			result.Value.Owner,
		)
	}

	if result.Value.Data == nil {
    	return nil, fmt.Errorf("escrow account data is missing")
	}

	// Decode account's binary data.
	data := result.Value.Data.GetBinary()

	return decodeEscrowState(data)
}

// decodeEscrowState decodes Anchor/Borsh serialized EscrowState data.
func decodeEscrowState(data []byte) (*EscrowState, error) {

	// Validate minimum account size.
	if len(data) < EscrowAccountSize {
		return nil, fmt.Errorf(
			"invalid escrow account size: expected at least %d bytes, got %d",
			EscrowAccountSize,
			len(data),
		)
	}

	// Validate Anchor account discrimiator.
	for i, expected := range escrowStateDiscriminator {
		if data[i] != expected {
			return nil, fmt.Errorf("invalid EscrowState discriminator")
		}
	}

	offset := EscrowDiscriminator

	// Decode client public key (32 bytes).
	var client solanaSDK.PublicKey
	copy(client[:], data[offset:offset+32])
	offset += 32

	// Decode freelancer public key (32 bytes).
	var freelancer solanaSDK.PublicKey
	copy(freelancer[:], data[offset:offset+32])
	offset += 32

	// Decode amount
	amount := binary.LittleEndian.Uint64(data[offset : offset + 8])
	offset += 8

	// Decode status (Borsh enum: one byte)
	status := Status(data[offset])
	offset++

	if status > StatusPending {
		return nil, fmt.Errorf("invalid escrow status: %d", status)
	}

	deadline := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Decode grace period (i64).
	gracePeriod := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Decode review window (i64).
	reviewWindow := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Decode completed_at (i64).
	completedAt := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Decode job_id (u64).
	jobID := binary.LittleEndian.Uint64(data[offset : offset+8])

	return &EscrowState{
		Client:       client,
		Freelancer:   freelancer,
		Amount:       amount,
		Status:       status,
		Deadline:     deadline,
		GracePeriod:  gracePeriod,
		ReviewWindow: reviewWindow,
		CompletedAt:  completedAt,
		JobID:        jobID,
	}, nil
}