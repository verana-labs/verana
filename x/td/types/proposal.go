package types

import (
	"cosmossdk.io/math"
	"strings"

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

func NewSlashTrustDepositProposal(title, description, account string, amount math.Int) *SlashTrustDepositProposal {
	return &SlashTrustDepositProposal{
		Title:       title,
		Description: description,
		Account:     account,
		Amount:      amount,
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

	// Validate account address
	if strings.TrimSpace(p.Account) == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("account cannot be empty")
	}

	_, err = sdk.AccAddressFromBech32(p.Account)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid account address: %v", err)
	}

	// amount must be > 0
	if p.Amount.IsNil() || !p.Amount.IsPositive() {
		return sdkerrors.ErrInvalidRequest.Wrap("amount must be positive")
	}

	return nil
}
