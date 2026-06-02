package lib

import (
	"context"
	"fmt"
	"log"
	"os"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	"github.com/verana-labs/verana/x/ec/types"
	permtypes "github.com/verana-labs/verana/x/pp/types"
)

// SendBankTransaction sends tokens from one account to another
func SendBankTransaction(client cosmosclient.Client, ctx context.Context, fromAddress, toAddress string, amount math.Int) error {
	account, err := client.Account(fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("amount...", amount)
	msg := banktypes.NewMsgSend(sdk.MustAccAddressFromBech32(fromAddress), sdk.MustAccAddressFromBech32(toAddress), sdk.NewCoins(sdk.NewCoin("uvna", amount)))

	txResp, err := client.BroadcastTx(ctx, account, msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("SendBankTransaction:\n\n")
	fmt.Println(txResp)

	return nil
}

// CreateEcosystem creates a new ecosystem.
// Spec draft 13: MsgCreateEcosystem seeds the ecosystem, an active v1
// governance framework version, AND the initial GF document from docURL +
// docHash in the ecosystem's default language.
func CreateEcosystem(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, did, docURL, docHash, language string) (string, error) {
	addr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	msg := &types.MsgCreateEcosystem{
		Corporation:  addr,
		Operator:     addr,
		Did:          did,
		Language:     language,
		DocUrl:       docURL,
		DocDigestSri: docHash,
	}

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("MsgCreateEcosystem:\n\n")
	fmt.Println(txResp)

	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		log.Fatal(err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "create_ecosystem" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "ecosystem_id" {
					fmt.Println("Created Ecosystem ID:", attribute.Value)
					return attribute.Value, nil
				}
			}
		}
	}
	return "no attribute found", fmt.Errorf("no attribute found")
}

// SubmitProposal submits a governance proposal
func SubmitProposal(client cosmosclient.Client, ctx context.Context, proposer cosmosaccount.Account, proposalFile string) error {
	proposalData, err := os.ReadFile(proposalFile)
	if err != nil {
		log.Fatal(err)
	}

	proposerAddr, err := proposer.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	content := &govtypes.TextProposal{
		Title:       "Proposal Title",
		Description: string(proposalData),
	}

	any, err := codectypes.NewAnyWithValue(content)
	if err != nil {
		log.Fatal(err)
	}

	msg := &govtypes.MsgSubmitProposal{
		Proposer: proposerAddr,
		Content:  any,
	}

	txResp, err := client.BroadcastTx(ctx, proposer, msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("SubmitProposal:\n\n")
	prettyJSON := PrettyJSON(client, txResp)
	fmt.Println(prettyJSON)

	return nil
}

// VoteOnProposal votes on a governance proposal
func VoteOnProposal(client cosmosclient.Client, ctx context.Context, voter cosmosaccount.Account, proposalID uint64, voteOption string) error {
	voterAddr, err := voter.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}
	msg := &govtypes.MsgVote{
		Voter:      voterAddr,
		ProposalId: proposalID,
		Option:     govtypes.VoteOption(govtypes.VoteOption_value[voteOption]),
	}

	txResp, err := client.BroadcastTx(ctx, voter, msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("VoteOnProposal:\n\n")
	prettyJSON := PrettyJSON(client, txResp)
	fmt.Println(prettyJSON)

	return nil
}

// CreateCredentialSchema creates a new credential schema
func CreateCredentialSchema(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, override cschema.MsgCreateCredentialSchema) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	msg := &cschema.MsgCreateCredentialSchema{
		Corporation: creatorAddr,
		Operator:    creatorAddr,
		EcosystemId: override.EcosystemId,
		JsonSchema:  override.JsonSchema,
	}

	var issuerGrantorValidity uint32 = 0
	var verifierGrantorValidity uint32 = 0
	var issuerValidity uint32 = 0
	var verifierValidity uint32 = 0
	var holderValidity uint32 = 0

	if override.IssuerGrantorValidationValidityPeriod != nil {
		issuerGrantorValidity = override.IssuerGrantorValidationValidityPeriod.Value
	}
	if override.VerifierGrantorValidationValidityPeriod != nil {
		verifierGrantorValidity = override.VerifierGrantorValidationValidityPeriod.Value
	}
	if override.IssuerValidationValidityPeriod != nil {
		issuerValidity = override.IssuerValidationValidityPeriod.Value
	}
	if override.VerifierValidationValidityPeriod != nil {
		verifierValidity = override.VerifierValidationValidityPeriod.Value
	}
	if override.HolderValidationValidityPeriod != nil {
		holderValidity = override.HolderValidationValidityPeriod.Value
	}

	msg.IssuerGrantorValidationValidityPeriod = &cschema.OptionalUInt32{Value: issuerGrantorValidity}
	msg.VerifierGrantorValidationValidityPeriod = &cschema.OptionalUInt32{Value: verifierGrantorValidity}
	msg.IssuerValidationValidityPeriod = &cschema.OptionalUInt32{Value: issuerValidity}
	msg.VerifierValidationValidityPeriod = &cschema.OptionalUInt32{Value: verifierValidity}
	msg.HolderValidationValidityPeriod = &cschema.OptionalUInt32{Value: holderValidity}

	msg.IssuerOnboardingMode = override.IssuerOnboardingMode
	msg.VerifierOnboardingMode = override.VerifierOnboardingMode
	msg.HolderOnboardingMode = override.HolderOnboardingMode

	msg.PricingAssetType = override.PricingAssetType
	msg.PricingAsset = override.PricingAsset
	msg.DigestAlgorithm = override.DigestAlgorithm

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("CreateCredentialSchema:\n\n")
	fmt.Println(txResp)

	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		log.Fatal(err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "create_credential_schema" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "credential_schema_id" {
					fmt.Println("Created CredentialSchema ID:", attribute.Value)
					return attribute.Value, nil
				}
			}
		}
	}
	return "no attribute found", fmt.Errorf("no attribute found")
}

// CreateRootPermission creates a root permission
func CreateRootPermission(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, override permtypes.MsgCreateRootParticipant) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	// [MOD-PP-MSG-7-3] spec v4 draft 13: handler hardcodes perm.type = ECOSYSTEM.
	_ = creatorAddr
	msg := &permtypes.MsgCreateRootParticipant{
		Corporation:      creatorAddr,
		Operator:         creatorAddr,
		SchemaId:         override.SchemaId,
		Did:              override.Did,
		EffectiveFrom:    override.EffectiveFrom,
		EffectiveUntil:   override.EffectiveUntil,
		ValidationFees:   override.ValidationFees,
		VerificationFees: override.VerificationFees,
		IssuanceFees:     override.IssuanceFees,
	}

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("CreatePermission:\n\n")
	fmt.Println(txResp)

	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		log.Fatal(err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "create_root_participant" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "root_participant_id" {
					fmt.Println("Created permission ID:", attribute.Value)
					return attribute.Value, nil
				}
			}
		}
	}
	return "no attribute found", fmt.Errorf("no attribute found")
}

// StartPermissionVP starts a permission validation process
func StartPermissionVP(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, override permtypes.MsgStartParticipantOP) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	msg := &permtypes.MsgStartParticipantOP{
		Corporation:                  override.Corporation,
		Operator:                     creatorAddr,
		Role:                         override.Role,
		Did:                          override.Did,
		ValidatorParticipantId:       override.ValidatorParticipantId,
		ValidationFees:               override.ValidationFees,
		IssuanceFees:                 override.IssuanceFees,
		VerificationFees:             override.VerificationFees,
		VsOperator:                   override.VsOperator,
		VsOperatorAuthzMsgTypes:      override.VsOperatorAuthzMsgTypes,
		VsOperatorAuthzSpendLimit:    override.VsOperatorAuthzSpendLimit,
		VsOperatorAuthzWithFeegrant:  override.VsOperatorAuthzWithFeegrant,
		VsOperatorAuthzFeeSpendLimit: override.VsOperatorAuthzFeeSpendLimit,
		VsOperatorAuthzPeriod:        override.VsOperatorAuthzPeriod,
	}
	if msg.Corporation == "" {
		msg.Corporation = creatorAddr
	}

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("StartPermissionVP:\n\n")
	fmt.Println(txResp)

	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		log.Fatal(err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "start_participant_op" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "participant_id" {
					fmt.Println("start permission ID:", attribute.Value)
					return attribute.Value, nil
				}
			}
		}
	}
	return "no attribute found", fmt.Errorf("no attribute found")
}

// SetPermissionVPToValidated sets a permission validation process to validated
func SetPermissionVPToValidated(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, override permtypes.MsgSetParticipantOPToValidated) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	msg := &permtypes.MsgSetParticipantOPToValidated{
		Corporation:             override.Corporation,
		Operator:                creatorAddr,
		Id:                      override.Id,
		ValidationFees:          override.ValidationFees,
		IssuanceFees:            override.IssuanceFees,
		VerificationFees:        override.VerificationFees,
		OpSummaryDigest:         override.OpSummaryDigest,
		IssuanceFeeDiscount:     override.IssuanceFeeDiscount,
		VerificationFeeDiscount: override.VerificationFeeDiscount,
	}
	if msg.Corporation == "" {
		msg.Corporation = creatorAddr
	}

	if override.EffectiveUntil != nil {
		msg.EffectiveUntil = override.EffectiveUntil
	}

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		return "", err
	}

	fmt.Print("SetPermissionVPToValidated:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}
