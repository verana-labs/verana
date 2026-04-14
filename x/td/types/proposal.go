package types

import (
	"strings"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	ProposalSlashTrustDeposit = "ProposalSlashTrustDeposit"
)

func init() {
	govtypes.RegisterProposalType(ProposalSlashTrustDeposit)
}

func NewSlashTrustDepositProposal(title, description, corporation string, deposit math.Int) *SlashTrustDepositProposal {
	return &SlashTrustDepositProposal{
		Title:       title,
		Description: description,
		Corporation: corporation,
		Deposit:     deposit,
	}
}

var _ govtypes.Content = &SlashTrustDepositProposal{}

func (p *SlashTrustDepositProposal) ProposalRoute() string { return RouterKey }

func (p *SlashTrustDepositProposal) ProposalType() string { return ProposalSlashTrustDeposit }

func (p *SlashTrustDepositProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(p)
	if err != nil {
		return err
	}

	// Validate corporation address
	if strings.TrimSpace(p.Corporation) == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("corporation cannot be empty")
	}

	_, err = sdk.AccAddressFromBech32(p.Corporation)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid corporation address: %v", err)
	}

	// deposit must be > 0
	if p.Deposit.IsNil() || !p.Deposit.IsPositive() {
		return sdkerrors.ErrInvalidRequest.Wrap("deposit must be positive")
	}

	return nil
}
