package solana

import (
	"crypto/sha256"

	solanaSDK "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
)

// BuildFundEscrowInstruction builds the Anchor's fund_escrow instruction
func (c *Client) BuildFundEscrowInstruction(
	client solanaSDK.PublicKey,
	escrow solanaSDK.PublicKey,
) solanaSDK.Instruction {

	// Generate the Anchor instruction discriminator.
	hash := sha256.Sum256([]byte("global:func_escrow"))
	data := hash[:8]

	// Define the accounts required by the FundEscrow instruction
	accounts := solanaSDK.AccountMetaSlice{
		solanaSDK.Meta(client).SIGNER().WRITE(),
		solanaSDK.Meta(escrow).WRITE(),
		solanaSDK.Meta(system.ProgramID),
	}

	// Create the instruction using the escrow program ID.
	return solanaSDK.NewInstruction(
		c.programID,
		accounts,
		data,
	)
}