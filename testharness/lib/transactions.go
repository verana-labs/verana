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
	didtypes "github.com/verana-labs/verana/x/dd/types"
	permtypes "github.com/verana-labs/verana/x/perm/types"
	"github.com/verana-labs/verana/x/tr/types"
)

// SendBankTransaction sends tokens from one account to another
func SendBankTransaction(client cosmosclient.Client, ctx context.Context, fromAddress, toAddress string, amount math.Int) error {
	// Get account from the keyring
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

	// Print response from broadcasting a transaction
	fmt.Print("SendBankTransaction:\n\n")
	fmt.Println(txResp)

	return nil
}

// CreateTrustRegistry creates a new trust registry
func CreateTrustRegistry(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, did, aka, docURL, docHash, language string) (string, error) {
	addr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	// Define a message to create a post
	msg := &types.MsgCreateTrustRegistry{
		Creator:      addr,
		Did:          did,
		Aka:          aka,
		Language:     language,
		DocUrl:       docURL,
		DocDigestSri: docHash,
	}

	// Broadcast a transaction from account `cooluser` with the message
	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("MsgCreateTrustRegistry:\n\n")
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
		if event.Type == "create_trust_registry" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "trust_registry_id" {
					fmt.Println("Created TrustRegistry ID:", attribute.Value)
					return attribute.Value, nil
				}
			}
		}
	}
	return "no attribute found", fmt.Errorf("no attribute found")
}

// SubmitProposal submits a governance proposal
func SubmitProposal(client cosmosclient.Client, ctx context.Context, proposer cosmosaccount.Account, proposalFile string) error {
	// Read proposal file
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

	// Print response from broadcasting a transaction
	fmt.Print("SubmitProposal:\n\n")
	prettyJSON := PrettyJSON(client, txResp)
	fmt.Println(prettyJSON)

	return nil
}

// VoteOnProposal votes on a governance proposal
func VoteOnProposal(client cosmosclient.Client, ctx context.Context, voter cosmosaccount.Account, proposalID uint64, voteOption string) error {
	// Create a vote message
	voterAddr, err := voter.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}
	msg := &govtypes.MsgVote{
		Voter:      voterAddr,
		ProposalId: proposalID,
		Option:     govtypes.VoteOption(govtypes.VoteOption_value[voteOption]),
	}

	// Broadcast the vote transaction
	txResp, err := client.BroadcastTx(ctx, voter, msg)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from broadcasting the transaction
	fmt.Print("VoteOnProposal:\n\n")
	prettyJSON := PrettyJSON(client, txResp)
	fmt.Println(prettyJSON)

	return nil
}

// AddDidToDirectory adds a DID to the directory
func AddDidToDirectory(client cosmosclient.Client, ctx context.Context, adder cosmosaccount.Account, did string, years uint32) error {
	creatorAddr, err := adder.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}
	msg := &didtypes.MsgAddDID{
		Creator: creatorAddr,
		Did:     did,
		Years:   uint32(years),
	}

	// Broadcast transaction to add DID
	txResp, err := client.BroadcastTx(ctx, adder, msg)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("AddDidToDirectory:\n\n")
	fmt.Println(txResp)

	return nil
}

// CreateCredentialSchema creates a new credential schema
func CreateCredentialSchema(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, override cschema.MsgCreateCredentialSchema) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	// Create message with all required fields
	// Validity period fields are mandatory - use 0 if not specified
	msg := &cschema.MsgCreateCredentialSchema{
		Creator:    creatorAddr,
		TrId:       override.TrId,
		JsonSchema: override.JsonSchema,
	}

	// Set validity period fields - use values from override if provided, otherwise 0
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

	// Always set the OptionalUInt32 fields (mandatory in new version)
	msg.IssuerGrantorValidationValidityPeriod = &cschema.OptionalUInt32{
		Value: issuerGrantorValidity,
	}
	msg.VerifierGrantorValidationValidityPeriod = &cschema.OptionalUInt32{
		Value: verifierGrantorValidity,
	}
	msg.IssuerValidationValidityPeriod = &cschema.OptionalUInt32{
		Value: issuerValidity,
	}
	msg.VerifierValidationValidityPeriod = &cschema.OptionalUInt32{
		Value: verifierValidity,
	}
	msg.HolderValidationValidityPeriod = &cschema.OptionalUInt32{
		Value: holderValidity,
	}

	// Set permission management modes
	msg.IssuerPermManagementMode = override.IssuerPermManagementMode
	msg.VerifierPermManagementMode = override.VerifierPermManagementMode

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from broadcasting a transaction
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
func CreateRootPermission(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, override permtypes.MsgCreateRootPermission) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	// Start with an empty struct and set only the defined attributes from override
	msg := &permtypes.MsgCreateRootPermission{
		Creator:          creatorAddr,
		SchemaId:         override.SchemaId,
		Did:              override.Did,
		EffectiveFrom:    override.EffectiveFrom,
		EffectiveUntil:   override.EffectiveUntil,
		ValidationFees:   override.ValidationFees,
		VerificationFees: override.VerificationFees,
		IssuanceFees:     override.IssuanceFees,
	}

	if override.Did != "" {
		msg.Did = override.Did
	}
	if override.Country != "" {
		msg.Country = override.Country
	}
	if override.ValidationFees != 0 {
		msg.ValidationFees = override.ValidationFees
	}
	if override.IssuanceFees != 0 {
		msg.IssuanceFees = override.IssuanceFees
	}
	if override.VerificationFees != 0 {
		msg.VerificationFees = override.VerificationFees
	}

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from broadcasting a transaction
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
		if event.Type == "create_root_permission" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "root_permission_id" {
					fmt.Println("Created permission ID:", attribute.Value)
					return attribute.Value, nil
				}
			}
		}
	}
	return "no attribute found", fmt.Errorf("no attribute found")
}

// StartPermissionVP starts a permission validation process
func StartPermissionVP(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, override permtypes.MsgStartPermissionVP) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	// Start with an empty struct and set only the defined attributes from override
	msg := &permtypes.MsgStartPermissionVP{
		Creator:          creatorAddr,
		Type:             override.Type,
		Did:              override.Did,
		ValidatorPermId:  override.ValidatorPermId,
		Country:          override.Country,
		ValidationFees:   override.ValidationFees,
		IssuanceFees:     override.IssuanceFees,
		VerificationFees: override.VerificationFees,
	}

	if override.Did != "" {
		msg.Did = override.Did
	}
	if override.Country != "" {
		msg.Country = override.Country
	}
	// Optional fee fields are already set from override

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		log.Fatal(err)
	}

	// Print response from broadcasting a transaction
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
		if event.Type == "start_permission_vp" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "permission_id" {
					fmt.Println("start permission ID:", attribute.Value)
					return attribute.Value, nil
				}
			}
		}
	}
	return "no attribute found", fmt.Errorf("no attribute found")
}

// SetPermissionVPToValidated sets a permission validation process to validated
func SetPermissionVPToValidated(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, override permtypes.MsgSetPermissionVPToValidated) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		log.Fatal(err)
	}

	// Create the message with proper creator address
	msg := &permtypes.MsgSetPermissionVPToValidated{
		Creator:            creatorAddr,
		Id:                 override.Id,
		ValidationFees:     override.ValidationFees,
		IssuanceFees:       override.IssuanceFees,
		VerificationFees:   override.VerificationFees,
		Country:            override.Country,
		VpSummaryDigestSri: override.VpSummaryDigestSri,
	}

	// Only set EffectiveUntil if it's explicitly provided
	// Let the blockchain calculate it otherwise
	if override.EffectiveUntil != nil {
		msg.EffectiveUntil = override.EffectiveUntil
	}

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		return "", err
	}

	// Print response from broadcasting a transaction
	fmt.Print("SetPermissionVPToValidated:\n\n")
	fmt.Println(txResp)

	// We don't need to extract any particular attribute here, just check success
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}
