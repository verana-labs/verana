package lib

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/group"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	protocolpooltypes "github.com/cosmos/cosmos-sdk/x/protocolpool/types"
	detypes "github.com/verana-labs/verana/x/de/types"
	didtypes "github.com/verana-labs/verana/x/dd/types"
	tdtypes "github.com/verana-labs/verana/x/td/types"
	trtypes "github.com/verana-labs/verana/x/tr/types"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	permtypes "github.com/verana-labs/verana/x/perm/types"
)

const (
	COOLUSER_ADDRESS                  = "verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4"
	FAUCET_ADDRESS                    = "verana167vrykn5vhp8v9rng69xf0jzvqa3v79etmr0t2"
	TRUST_REGISTRY_CONTROLLER_ADDRESS = "verana12dyk649yce4dvdppehsyraxe6p6jemzg2qwutf"
	ISSUER_GRANTOR_APPLICANT_ADDRESS  = "verana10gcacdzdv6hw6qfyq93kcqkxrhcxc077ap9kmz"
	ISSUER_APPLICANT_ADDRESS          = "verana1gnt6yu57zalyml8kumcypdvtdh28fvzfuqrn9l"
	VERIFIER_APPLICANT_ADDRESS        = "verana1rx88fyxcrcpzx7v02aln0vks4c0dtnuy2yldlw"
	CREDENTIAL_HOLDER_ADDRESS         = "verana15tc85w6wxmwemm7ytwxhkglm8rzsnmpkr4keer"
)

const (
	COOLUSER_NAME                  = "cooluser"
	TRUST_REGISTRY_CONTROLLER_NAME = "Trust_Registry_Controller"
	ISSUER_GRANTOR_APPLICANT_NAME  = "Issuer_Grantor_Applicant"
	ISSUER_APPLICANT_NAME          = "Issuer_Applicant"
	VERIFIER_APPLICANT_NAME        = "Verifier_Applicant"
	CREDENTIAL_HOLDER_NAME         = "Credential_Holder"
)

// GenerateUniqueDID generates a unique DID
// GenerateUniqueDID generates a unique DID with proper randomness
func GenerateUniqueDID(client cosmosclient.Client, ctx context.Context) string {
	listRegs, err := ListTrustRegistries(client, ctx, 1000)
	if err != nil {
		panic(fmt.Sprintf("Failed to list trust registries: %v", err))
	}

	// Create a more random base by including timestamp and random bytes
	baseDid := "did:example:"

	// Add current timestamp to make it more unique
	timestamp := time.Now().UnixNano()

	// Generate 8 random bytes
	randomBytes := make([]byte, 8)
	_, err = rand.Read(randomBytes)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate random bytes: %v", err))
	}

	// Create a unique identifier by combining timestamp and random bytes
	uniqueID := fmt.Sprintf("%x%x", timestamp, randomBytes)

	// Ensure the DID doesn't already exist
	did := baseDid + uniqueID
	if IsDidUsed(listRegs, did) {
		// In the extremely unlikely case of a collision, try again with more randomness
		time.Sleep(time.Millisecond) // Ensure different timestamp
		return GenerateUniqueDID(client, ctx)
	}

	return did
}

// CreateNewTrustRegistry creates a new trust registry and returns its ID
func CreateNewTrustRegistry(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, did string) string {
	trIdStr, err := CreateTrustRegistry(client,
		ctx,
		account,
		did,
		"http://example-aka.com",
		"https://example.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en")
	if err != nil {
		panic(fmt.Sprintf("Failed to create trust registry: %v", err))
	}
	return trIdStr
}

// CreateSimpleCredentialSchema creates a credential schema and returns its ID
func CreateSimpleCredentialSchema(
	client cosmosclient.Client,
	ctx context.Context,
	account cosmosaccount.Account,
	trustRegistryID string,
	schemaData string,
	issuerMode cschema.CredentialSchemaPermManagementMode,
	verifierMode cschema.CredentialSchemaPermManagementMode,
) string {
	trId, err := strconv.ParseUint(trustRegistryID, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse trust registry ID: %v", err))
	}

	// Create credential schema with default validity periods (0 = never expire)
	csStrId, err := CreateCredentialSchema(client, ctx, account, cschema.MsgCreateCredentialSchema{
		TrId:                       trId,
		JsonSchema:                 schemaData,
		IssuerPermManagementMode:   uint32(issuerMode),
		VerifierPermManagementMode: uint32(verifierMode),
		// Validity periods are mandatory - use 0 (never expire) as default
		IssuerGrantorValidationValidityPeriod:   &cschema.OptionalUInt32{Value: 0},
		VerifierGrantorValidationValidityPeriod: &cschema.OptionalUInt32{Value: 0},
		IssuerValidationValidityPeriod:          &cschema.OptionalUInt32{Value: 0},
		VerifierValidationValidityPeriod:        &cschema.OptionalUInt32{Value: 0},
		HolderValidationValidityPeriod:          &cschema.OptionalUInt32{Value: 0},
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create credential schema: %v", err))
	}
	return csStrId
}

// CreateRootPermissionWithDates creates a root permission with specific dates
func CreateRootPermissionWithDates(
	client cosmosclient.Client,
	ctx context.Context,
	account cosmosaccount.Account,
	schemaID string,
	did string,
	effectiveFrom time.Time,
	effectiveUntil time.Time,
	validationFees uint64,
	verificationFees uint64,
	issuanceFees uint64,
) string {
	csId, err := strconv.ParseUint(schemaID, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse schema ID: %v", err))
	}

	rpStrId, err := CreateRootPermission(client, ctx, account, permtypes.MsgCreateRootPermission{
		SchemaId:         csId,
		Did:              did,
		EffectiveFrom:    &effectiveFrom,
		EffectiveUntil:   &effectiveUntil,
		ValidationFees:   validationFees,
		VerificationFees: verificationFees,
		IssuanceFees:     issuanceFees,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create root permission: %v", err))
	}
	return rpStrId
}

// RegisterDID adds a DID to the directory
func RegisterDID(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, did string, years uint32) {
	err := AddDidToDirectory(client, ctx, account, did, years)
	if err != nil {
		panic(fmt.Sprintf("Failed to add DID to directory: %v", err))
	}
}

// SaveJourneyResult saves a journey result for later use
func SaveJourneyResult(journeyName string, result interface{}) error {
	// Create a directory for results if it doesn't exist
	err := os.MkdirAll("journey_results", 0755)
	if err != nil {
		return fmt.Errorf("failed to create journey_results directory: %v", err)
	}

	// Marshal the result to JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %v", err)
	}

	// Write the result to a file
	filename := fmt.Sprintf("journey_results/%s.json", journeyName)
	err = os.WriteFile(filename, resultJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write result to file: %v", err)
	}

	return nil
}

// GetJourneyResult retrieves a journey result
func GetJourneyResult(journeyName string) (JourneyResult, error) {
	var result JourneyResult

	// Read the result from the file
	filename := fmt.Sprintf("journey_results/%s.json", journeyName)
	resultJSON, err := os.ReadFile(filename)
	if err != nil {
		return result, fmt.Errorf("failed to read result from file: %v", err)
	}

	// Unmarshal the result from JSON
	err = json.Unmarshal(resultJSON, &result)
	if err != nil {
		return result, fmt.Errorf("failed to unmarshal result: %v", err)
	}

	return result, nil
}

// GetAccount gets an account by name, panicking on error
func GetAccount(client cosmosclient.Client, name string) cosmosaccount.Account {
	account, err := client.Account(name)
	if err != nil {
		panic(fmt.Sprintf("Failed to get account %s: %v", name, err))
	}
	return account
}

// SendFunds sends funds from one account to another, panicking on error
func SendFunds(client cosmosclient.Client, ctx context.Context, fromAddress, toAddress string, amount math.Int) {
	err := SendBankTransaction(client, ctx, fromAddress, toAddress, amount)
	if err != nil {
		panic(fmt.Sprintf("Failed to send funds from %s to %s: %v", fromAddress, toAddress, err))
	}
}

// LoadJourneyResult loads a journey result, panicking on error
func LoadJourneyResult(journeyName string) JourneyResult {
	result, err := GetJourneyResult(journeyName)
	if err != nil {
		panic(fmt.Sprintf("Failed to load %s results: %v", journeyName, err))
	}
	return result
}

// StartValidationProcess starts a permission validation process, panicking on error
func StartValidationProcess(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, msg permtypes.MsgStartPermissionVP) string {
	permissionID, err := StartPermissionVP(client, ctx, account, msg)
	if err != nil {
		panic(fmt.Sprintf("Failed to start permission validation: %v", err))
	}
	return permissionID
}

// ValidatePermission validates a permission, panicking on error
func ValidatePermission(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, permID, validationFees, issuanceFees, verificationFees uint64, country string) {
	ValidatePermissionWithDiscounts(client, ctx, account, permID, validationFees, issuanceFees, verificationFees, country, 0, 0)
}

// ValidatePermissionWithDiscounts validates a permission with optional fee discounts, panicking on error
func ValidatePermissionWithDiscounts(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, permID, validationFees, issuanceFees, verificationFees uint64, country string, issuanceFeeDiscount, verificationFeeDiscount uint64) {
	validateMsg := permtypes.MsgSetPermissionVPToValidated{
		Id:                      permID,
		ValidationFees:          validationFees,
		IssuanceFees:            issuanceFees,
		VerificationFees:        verificationFees,
		Country:                 country,
		IssuanceFeeDiscount:     issuanceFeeDiscount,
		VerificationFeeDiscount: verificationFeeDiscount,
	}

	_, err := SetPermissionVPToValidated(client, ctx, account, validateMsg)
	if err != nil {
		panic(fmt.Sprintf("Failed to set permission to validated: %v", err))
	}
}

// VerifyPendingValidation verifies a permission is in pending validation state
func VerifyPendingValidation(client cosmosclient.Client, ctx context.Context, permID uint64, expectedDID string, expectedType string) bool {
	resp, err := QueryPermission(client, ctx, permID)
	if err != nil {
		fmt.Printf("❌ Permission validation verification failed: %v\n", err)
		return false
	}

	// Verify permission is in PENDING state
	if resp.Permission.VpState != permtypes.ValidationState_PENDING {
		fmt.Printf("❌ Permission validation verification failed: Expected state PENDING, got %s\n",
			resp.Permission.VpState)
		return false
	}

	// Verify DID and type match expectations
	permType := permtypes.PermissionType_name[int32(resp.Permission.Type)]
	if permType != expectedType {
		fmt.Printf("❌ Permission validation verification failed: Expected type %s, got %s\n",
			expectedType, permType)
		return false
	}

	if resp.Permission.Did != expectedDID {
		fmt.Printf("❌ Permission validation verification failed: Expected DID %s, got %s\n",
			expectedDID, resp.Permission.Did)
		return false
	}

	fmt.Printf("✅ Verified permission ID %d in PENDING state with expected type %s and DID %s\n",
		permID, permType, resp.Permission.Did)
	return true
}

// VerifyValidatedPermission verifies a permission is properly validated with expected values
func VerifyValidatedPermission(client cosmosclient.Client, ctx context.Context, permID uint64,
	expectedDID, expectedType string,
	expectedValidationFees, expectedIssuanceFees, expectedVerificationFees uint64) bool {

	resp, err := QueryPermission(client, ctx, permID)
	if err != nil {
		fmt.Printf("❌ Validated permission verification failed: %v\n", err)
		return false
	}

	// Verify permission is in VALIDATED state
	if resp.Permission.VpState != permtypes.ValidationState_VALIDATED {
		fmt.Printf("❌ Validated permission verification failed: Expected state VALIDATED, got %s\n",
			resp.Permission.VpState)
		return false
	}

	// Verify type matches expectation
	permType := permtypes.PermissionType_name[int32(resp.Permission.Type)]
	if permType != expectedType {
		fmt.Printf("❌ Validated permission verification failed: Expected type %s, got %s\n",
			expectedType, permType)
		return false
	}

	// Verify DID matches expectation
	if resp.Permission.Did != expectedDID {
		fmt.Printf("❌ Validated permission verification failed: Expected DID %s, got %s\n",
			expectedDID, resp.Permission.Did)
		return false
	}

	// Verify fees match expectations
	if resp.Permission.ValidationFees != expectedValidationFees {
		fmt.Printf("❌ Validated permission verification failed: Expected validation fees %d, got %d\n",
			expectedValidationFees, resp.Permission.ValidationFees)
		return false
	}

	if resp.Permission.IssuanceFees != expectedIssuanceFees {
		fmt.Printf("❌ Validated permission verification failed: Expected issuance fees %d, got %d\n",
			expectedIssuanceFees, resp.Permission.IssuanceFees)
		return false
	}

	if resp.Permission.VerificationFees != expectedVerificationFees {
		fmt.Printf("❌ Validated permission verification failed: Expected verification fees %d, got %d\n",
			expectedVerificationFees, resp.Permission.VerificationFees)
		return false
	}

	fmt.Printf("✅ Verified permission ID %d is VALIDATED with correct type %s, DID %s, and fees\n",
		permID, permType, resp.Permission.Did)
	return true
}

// GetAddressPrefix Get address prefix
func GetAddressPrefix() string {
	return addressPrefix
}

// GetPermission retrieves a permission by ID
func GetPermission(client cosmosclient.Client, ctx context.Context, permID uint64) (*permtypes.Permission, error) {
	permClient := permtypes.NewQueryClient(client.Context())
	resp, err := permClient.GetPermission(ctx, &permtypes.QueryGetPermissionRequest{
		Id: permID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query permission: %v", err)
	}

	return &resp.Permission, nil
}

// RenewPermissionVP initiates the renewal of a permission validation process
func RenewPermissionVP(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, msg permtypes.MsgRenewPermissionVP) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %v", err)
	}

	// Ensure creator is set correctly
	msgWithCreator := permtypes.MsgRenewPermissionVP{
		Creator: creatorAddr,
		Id:      msg.Id,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast renewal transaction: %v", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("RenewPermissionVP:\n\n")
	fmt.Println(txResp)

	// We're using the same permission ID for renewal
	return fmt.Sprintf("%d", msg.Id), nil
}

// GetTrustDeposit gets the trust deposit for an account
func GetTrustDeposit(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account) (*tdtypes.TrustDeposit, error) {
	return GetTrustDepositAtHeight(client, ctx, account, 0)
}

// GetTrustDepositAtHeight gets the trust deposit for an account at a specific block height
// If height is 0, queries at latest height
func GetTrustDepositAtHeight(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, height int64) (*tdtypes.TrustDeposit, error) {
	creatorAddr, err := account.Address(addressPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get account address: %v", err)
	}

	tdClient := tdtypes.NewQueryClient(client.Context())
	req := &tdtypes.QueryGetTrustDepositRequest{
		Account: creatorAddr,
	}

	// For height-specific queries, we need to use command line with --height flag
	if height > 0 {
		cmd := exec.Command("veranad", "q", "td", "get-trust-deposit", creatorAddr, "--height", fmt.Sprintf("%d", height), "-o", "json")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to query trust deposit at height %d: %v", height, err)
		}

		var resp struct {
			TrustDeposit tdtypes.TrustDeposit `json:"trustDeposit"`
		}

		if err := json.Unmarshal(output, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse trust deposit JSON at height %d: %v", height, err)
		}

		return &resp.TrustDeposit, nil
	}

	// Use gRPC query for latest height
	resp, err := tdClient.GetTrustDeposit(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query trust deposit: %v", err)
	}

	return &resp.TrustDeposit, nil
}

// ReclaimTrustDeposit reclaims trust deposit
func ReclaimTrustDeposit(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, msg tdtypes.MsgReclaimTrustDeposit) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %v", err)
	}

	// Ensure creator is set correctly
	msgWithCreator := tdtypes.MsgReclaimTrustDeposit{
		Creator: creatorAddr,
		Claimed: msg.Claimed,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast reclaim trust deposit: %v", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("ReclaimTrustDeposit:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// ReclaimTrustDepositYield reclaims trust deposit yield
func ReclaimTrustDepositYield(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account) (string, error) {
	_, _, err := ReclaimTrustDepositYieldWithResponse(client, ctx, creator)
	if err != nil {
		return "", err
	}
	return "success", nil
}

// ReclaimTrustDepositYieldWithResponse reclaims trust deposit yield and returns the response
// Returns the claimed amount and the block height where the transaction was executed
func ReclaimTrustDepositYieldWithResponse(client cosmosclient.Client, ctx context.Context,
	creator cosmosaccount.Account) (*tdtypes.MsgReclaimTrustDepositYieldResponse, int64, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get creator address: %v", err)
	}

	msg := tdtypes.MsgReclaimTrustDepositYield{
		Creator: creatorAddr,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msg)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to broadcast reclaim trust deposit yield: %v", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("ReclaimTrustDepositYield:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return nil, 0, fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	// Extract block height from transaction response
	blockHeight := txResp.TxResponse.Height

	// Extract the response from events
	// Event type can be either "verana.td.v1.EventReclaimTrustDepositYield" or "reclaim_trust_deposit_yield"
	claimedAmount := uint64(0)
	for _, event := range txResp.TxResponse.Events {
		if event.Type == "verana.td.v1.EventReclaimTrustDepositYield" || event.Type == "reclaim_trust_deposit_yield" {
			for _, attr := range event.Attributes {
				if attr.Key == "claimed_yield" {
					if val, parseErr := strconv.ParseUint(attr.Value, 10, 64); parseErr == nil {
						claimedAmount = val
					}
				}
			}
		}
	}

	response := &tdtypes.MsgReclaimTrustDepositYieldResponse{
		ClaimedAmount: claimedAmount,
	}

	return response, blockHeight, nil
}

// GetTrustDepositParams gets the trust deposit module parameters
func GetTrustDepositParams(client cosmosclient.Client, ctx context.Context) (*tdtypes.Params, error) {
	return GetTrustDepositParamsAtHeight(client, ctx, 0)
}

// GetTrustDepositParamsAtHeight gets the trust deposit module parameters at a specific block height
// If height is 0, queries at latest height
func GetTrustDepositParamsAtHeight(client cosmosclient.Client, ctx context.Context, height int64) (*tdtypes.Params, error) {
	// For height-specific queries, use command line with --height flag
	if height > 0 {
		cmd := exec.Command("veranad", "q", "td", "params", "--height", fmt.Sprintf("%d", height), "-o", "json")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to query trust deposit params at height %d: %v", height, err)
		}

		var resp struct {
			Params tdtypes.Params `json:"params"`
		}

		if err := json.Unmarshal(output, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse trust deposit params JSON at height %d: %v", height, err)
		}

		return &resp.Params, nil
	}

	// Use gRPC query for latest height
	tdClient := tdtypes.NewQueryClient(client.Context())
	resp, err := tdClient.Params(ctx, &tdtypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query trust deposit parameters: %v", err)
	}

	return &resp.Params, nil
}

// GetBankBalance gets the bank balance for an address
func GetBankBalance(client cosmosclient.Client, ctx context.Context, address string) (sdk.Coin, error) {
	return GetBankBalanceAtHeight(client, ctx, address, 0)
}

// GetBankBalanceAtHeight gets the bank balance for an address at a specific block height
// If height is 0, queries at latest height
func GetBankBalanceAtHeight(client cosmosclient.Client, ctx context.Context, address string, height int64) (sdk.Coin, error) {
	bankClient := banktypes.NewQueryClient(client.Context())
	req := &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   "uvna",
	}

	// Set height if specified (height > 0)
	if height > 0 {
		// Use command line query with --height flag for deterministic queries
		cmd := exec.Command("veranad", "q", "bank", "balances", address, "--height", fmt.Sprintf("%d", height), "-o", "json")
		output, err := cmd.Output()
		if err != nil {
			return sdk.Coin{}, fmt.Errorf("failed to query bank balance at height %d: %v", height, err)
		}

		var balancesResp struct {
			Balances []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"balances"`
		}

		if err := json.Unmarshal(output, &balancesResp); err != nil {
			return sdk.Coin{}, fmt.Errorf("failed to parse bank balance JSON at height %d: %v", height, err)
		}

		// Find uvna balance
		for _, bal := range balancesResp.Balances {
			if bal.Denom == "uvna" {
				amount, ok := math.NewIntFromString(bal.Amount)
				if !ok {
					return sdk.Coin{}, fmt.Errorf("failed to parse amount: %s", bal.Amount)
				}
				return sdk.NewCoin("uvna", amount), nil
			}
		}

		return sdk.NewCoin("uvna", math.ZeroInt()), nil
	}

	// Use gRPC query for latest height
	resp, err := bankClient.Balance(ctx, req)
	if err != nil {
		return sdk.Coin{}, fmt.Errorf("failed to query bank balance: %v", err)
	}

	if resp.Balance == nil {
		return sdk.NewCoin("uvna", math.ZeroInt()), nil
	}

	return *resp.Balance, nil
}

// VerifyPermissionTerminationRequested verifies a permission is in TERMINATION_REQUESTED state
func VerifyPermissionTerminationRequested(client cosmosclient.Client, ctx context.Context, permID uint64) bool {
	perm, err := GetPermission(client, ctx, permID)
	if err != nil {
		fmt.Printf("❌ Permission termination verification failed: %v\n", err)
		return false
	}

	if perm.VpState != permtypes.ValidationState_TERMINATION_REQUESTED {
		fmt.Printf("❌ Permission termination verification failed: Permission not in TERMINATION_REQUESTED state, current state: %s\n",
			perm.VpState)
		return false
	}

	if perm.VpTermRequested == nil {
		fmt.Printf("❌ Permission termination verification failed: VpTermRequested timestamp is not set\n")
		return false
	}

	fmt.Printf("✅ Verified permission ID %d is in TERMINATION_REQUESTED state\n", permID)
	return true
}

// VerifyPermissionTerminated verifies a permission is in TERMINATED state
func VerifyPermissionTerminated(client cosmosclient.Client, ctx context.Context, permID uint64) bool {
	perm, err := GetPermission(client, ctx, permID)
	if err != nil {
		fmt.Printf("❌ Permission termination verification failed: %v\n", err)
		return false
	}

	if perm.VpState != permtypes.ValidationState_TERMINATED {
		fmt.Printf("❌ Permission termination verification failed: Permission not in TERMINATED state, current state: %s\n",
			perm.VpState)
		return false
	}

	fmt.Printf("✅ Verified permission ID %d is in TERMINATED state\n", permID)
	return true
}

// VerifyTrustDepositClaimable verifies a trust deposit has been made claimable
func VerifyTrustDepositClaimable(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, initialDeposit uint64) bool {
	trustDeposit, err := GetTrustDeposit(client, ctx, account)
	if err != nil {
		fmt.Printf("❌ Trust deposit verification failed: %v\n", err)
		return false
	}

	if initialDeposit > 0 && trustDeposit.Claimable == 0 {
		fmt.Printf("❌ Trust deposit verification failed: Initial deposit was %d but claimable deposit is 0\n", initialDeposit)
		return false
	}

	fmt.Printf("✅ Verified trust deposit is claimable: %d\n", trustDeposit.Claimable)
	return true
}

// VerifyTrustDepositReclaimed verifies a trust deposit has been reclaimed
func VerifyTrustDepositReclaimed(client cosmosclient.Client, ctx context.Context, account cosmosaccount.Account, beforeClaimable uint64) bool {
	trustDeposit, err := GetTrustDeposit(client, ctx, account)
	if err != nil {
		fmt.Printf("❌ Trust deposit reclaim verification failed: %v\n", err)
		return false
	}

	if trustDeposit.Claimable >= beforeClaimable {
		fmt.Printf("❌ Trust deposit reclaim verification failed: Claimable amount not reduced. Before: %d, After: %d\n",
			beforeClaimable, trustDeposit.Claimable)
		return false
	}

	fmt.Printf("✅ Verified trust deposit was successfully reclaimed. New claimable amount: %d\n", trustDeposit.Claimable)
	return true
}

// AddGovernanceFrameworkDocument adds a new governance framework document to a trust registry
func AddGovernanceFrameworkDocument(
	client cosmosclient.Client,
	ctx context.Context,
	creator cosmosaccount.Account,
	msg trtypes.MsgAddGovernanceFrameworkDocument,
) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %v", err)
	}

	// Create the complete message with authority and operator addresses
	// For v4 spec, authority and operator are both the creator's address
	msgWithCreator := trtypes.MsgAddGovernanceFrameworkDocument{
		Authority:    creatorAddr,
		Operator:     creatorAddr,
		Id:           msg.Id,
		DocLanguage:  msg.DocLanguage,
		DocUrl:       msg.DocUrl,
		DocDigestSri: msg.DocDigestSri,
		Version:      msg.Version,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %v", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("AddGovernanceFrameworkDocument:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// IncreaseActiveGovernanceFrameworkVersion increases the active version of a trust registry's governance framework
func IncreaseActiveGovernanceFrameworkVersion(
	client cosmosclient.Client,
	ctx context.Context,
	creator cosmosaccount.Account,
	msg trtypes.MsgIncreaseActiveGovernanceFrameworkVersion,
) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %v", err)
	}

	// Create the complete message with authority and operator addresses
	// For v4 spec, authority and operator are both the creator's address
	msgWithCreator := trtypes.MsgIncreaseActiveGovernanceFrameworkVersion{
		Authority: creatorAddr,
		Operator:  creatorAddr,
		Id:        msg.Id,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %v", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("IncreaseActiveGovernanceFrameworkVersion:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// VerifyGovernanceFrameworkUpdate verifies that a trust registry's governance framework was updated correctly
func VerifyGovernanceFrameworkUpdate(
	client cosmosclient.Client,
	ctx context.Context,
	trustRegistryID uint64,
	expectedActiveVersion uint32,
) bool {
	trustRegistry, err := QueryTrustRegistry(client, ctx, trustRegistryID)
	if err != nil {
		fmt.Printf("❌ Governance framework update verification failed: %v\n", err)
		return false
	}

	if trustRegistry.TrustRegistry.ActiveVersion != int32(expectedActiveVersion) {
		fmt.Printf("❌ Governance framework update verification failed: Expected active version %d, got %d\n",
			expectedActiveVersion, trustRegistry.TrustRegistry.ActiveVersion)
		return false
	}

	// Verify that versions array includes the new version
	versionFound := false
	for _, version := range trustRegistry.TrustRegistry.Versions {
		if version.Version == int32(expectedActiveVersion) {
			versionFound = true
			break
		}
	}

	if !versionFound {
		fmt.Printf("❌ Governance framework update verification failed: Version %d not found in versions list\n",
			expectedActiveVersion)
		return false
	}

	fmt.Printf("✅ Verified governance framework updated to version %d\n", expectedActiveVersion)
	return true
}

// GetDID retrieves a DID entry from the DID Directory
func GetDID(client cosmosclient.Client, ctx context.Context, did string) (*didtypes.DIDDirectory, error) {
	didClient := didtypes.NewQueryClient(client.Context())
	resp, err := didClient.GetDID(ctx, &didtypes.QueryGetDIDRequest{
		Did: did,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query DID: %w", err)
	}
	return &resp.Did, nil
}

// RenewDID renews a DID in the DID Directory
func RenewDID(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, msg didtypes.MsgRenewDID) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %w", err)
	}

	// Make sure creator is set correctly
	msgWithCreator := didtypes.MsgRenewDID{
		Creator: creatorAddr,
		Did:     msg.Did,
		Years:   msg.Years,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("RenewDID:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// TouchDID updates the indexing of a DID in the DID Directory
func TouchDID(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, msg didtypes.MsgTouchDID) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %w", err)
	}

	// Make sure creator is set correctly
	msgWithCreator := didtypes.MsgTouchDID{
		Creator: creatorAddr,
		Did:     msg.Did,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("TouchDID:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// RemoveDID removes a DID from the DID Directory
func RemoveDID(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, msg didtypes.MsgRemoveDID) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %w", err)
	}

	// Make sure creator is set correctly
	msgWithCreator := didtypes.MsgRemoveDID{
		Creator: creatorAddr,
		Did:     msg.Did,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("RemoveDID:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// RevokePermission revokes a permission
func RevokePermission(client cosmosclient.Client, ctx context.Context, validator cosmosaccount.Account, msg permtypes.MsgRevokePermission) (string, error) {
	validatorAddr, err := validator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get validator address: %w", err)
	}

	// Make sure creator is set correctly
	msgWithCreator := permtypes.MsgRevokePermission{
		Creator: validatorAddr,
		Id:      msg.Id,
	}

	txResp, err := client.BroadcastTx(ctx, validator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("RevokePermission:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// ExtendPermission extends the validity period of a permission
func ExtendPermission(client cosmosclient.Client, ctx context.Context, validator cosmosaccount.Account, msg permtypes.MsgExtendPermission) (string, error) {
	validatorAddr, err := validator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get validator address: %w", err)
	}

	// Make sure creator is set correctly
	msgWithCreator := permtypes.MsgExtendPermission{
		Creator:        validatorAddr,
		Id:             msg.Id,
		EffectiveUntil: msg.EffectiveUntil,
	}

	txResp, err := client.BroadcastTx(ctx, validator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("ExtendPermission:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// UpdateCredentialSchema updates an existing credential schema with new validation periods
func UpdateCredentialSchema(
	client cosmosclient.Client,
	ctx context.Context,
	creator cosmosaccount.Account,
	schemaID uint64,
	issuerGrantorValidationValidityPeriod uint32,
	verifierGrantorValidationValidityPeriod uint32,
	issuerValidationValidityPeriod uint32,
	verifierValidationValidityPeriod uint32,
	holderValidationValidityPeriod uint32,
) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %w", err)
	}

	// Create complete message with creator address
	// All validity period fields are mandatory - always set them (use 0 if not updating)
	msgWithCreator := cschema.MsgUpdateCredentialSchema{
		Creator: creatorAddr,
		Id:      schemaID,
		// Always set OptionalUInt32 fields (mandatory in new version)
		IssuerGrantorValidationValidityPeriod: &cschema.OptionalUInt32{
			Value: issuerGrantorValidationValidityPeriod,
		},
		VerifierGrantorValidationValidityPeriod: &cschema.OptionalUInt32{
			Value: verifierGrantorValidationValidityPeriod,
		},
		IssuerValidationValidityPeriod: &cschema.OptionalUInt32{
			Value: issuerValidationValidityPeriod,
		},
		VerifierValidationValidityPeriod: &cschema.OptionalUInt32{
			Value: verifierValidationValidityPeriod,
		},
		HolderValidationValidityPeriod: &cschema.OptionalUInt32{
			Value: holderValidationValidityPeriod,
		},
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("UpdateCredentialSchema:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// ArchiveCredentialSchema archives or unarchives a credential schema
func ArchiveCredentialSchema(client cosmosclient.Client, ctx context.Context, creator cosmosaccount.Account, msg cschema.MsgArchiveCredentialSchema) (string, error) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get creator address: %w", err)
	}

	// Create complete message with creator address
	msgWithCreator := cschema.MsgArchiveCredentialSchema{
		Creator: creatorAddr,
		Id:      msg.Id,
		Archive: msg.Archive,
	}

	txResp, err := client.BroadcastTx(ctx, creator, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("ArchiveCredentialSchema:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	if msg.Archive {
		return "archived", nil
	}
	return "unarchived", nil
}

func CancelPermissionVPLastRequest(client cosmosclient.Client, ctx context.Context, applicant cosmosaccount.Account, msg permtypes.MsgCancelPermissionVPLastRequest) (string, error) {
	applicantAddr, err := applicant.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get applicant address: %w", err)
	}

	// Create complete message with creator address
	msgWithCreator := permtypes.MsgCancelPermissionVPLastRequest{
		Creator: applicantAddr,
		Id:      msg.Id,
	}

	txResp, err := client.BroadcastTx(ctx, applicant, &msgWithCreator)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("CancelPermissionVPLastRequest:\n\n")
	fmt.Println(txResp)

	// Check if the transaction was successful
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return "success", nil
}

// SlashPermissionTrustDeposit slashes a permission's trust deposit
func SlashPermissionTrustDeposit(client cosmosclient.Client, ctx context.Context, actor cosmosaccount.Account, msg permtypes.MsgSlashPermissionTrustDeposit) string {
	actorAddr, err := actor.Address(addressPrefix)
	if err != nil {
		panic(fmt.Sprintf("Failed to get actor address: %v", err))
	}

	msgWithCreator := permtypes.MsgSlashPermissionTrustDeposit{
		Creator: actorAddr,
		Id:      msg.Id,
		Amount:  msg.Amount,
	}

	txResp, err := client.BroadcastTx(ctx, actor, &msgWithCreator)
	if err != nil {
		panic(fmt.Sprintf("Failed to broadcast slash permission trust deposit: %v", err))
	}

	fmt.Print("SlashPermissionTrustDeposit:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		panic(fmt.Sprintf("transaction failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog))
	}

	return "success"
}

// RepayPermissionSlashedTrustDeposit repays a slashed permission's trust deposit
func RepayPermissionSlashedTrustDeposit(client cosmosclient.Client, ctx context.Context, actor cosmosaccount.Account, msg permtypes.MsgRepayPermissionSlashedTrustDeposit) string {
	actorAddr, err := actor.Address(addressPrefix)
	if err != nil {
		panic(fmt.Sprintf("Failed to get actor address: %v", err))
	}

	msgWithCreator := permtypes.MsgRepayPermissionSlashedTrustDeposit{
		Creator: actorAddr,
		Id:      msg.Id,
	}

	txResp, err := client.BroadcastTx(ctx, actor, &msgWithCreator)
	if err != nil {
		panic(fmt.Sprintf("Failed to broadcast repay permission slashed trust deposit: %v", err))
	}

	fmt.Print("RepayPermissionSlashedTrustDeposit:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		panic(fmt.Sprintf("transaction failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog))
	}

	return "success"
}

// CreatePermission creates a permission directly
func CreatePermission(client cosmosclient.Client, ctx context.Context, actor cosmosaccount.Account, msg permtypes.MsgCreatePermission) string {
	actorAddr, err := actor.Address(addressPrefix)
	if err != nil {
		panic(fmt.Sprintf("Failed to get actor address: %v", err))
	}

	msgWithCreator := permtypes.MsgCreatePermission{
		Creator:          actorAddr,
		SchemaId:         msg.SchemaId,
		Type:             msg.Type,
		Did:              msg.Did,
		Country:          msg.Country,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		VerificationFees: msg.VerificationFees,
	}

	txResp, err := client.BroadcastTx(ctx, actor, &msgWithCreator)
	if err != nil {
		panic(fmt.Sprintf("Failed to broadcast create permission: %v", err))
	}

	fmt.Print("CreatePermission:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		panic(fmt.Sprintf("transaction failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog))
	}

	return "success"
}

// CreatePermissionSession creates a new permission session
func CreatePermissionSession(
	client cosmosclient.Client,
	ctx context.Context,
	creator cosmosaccount.Account,
	sessionID string,
	issuerPermID uint64,
	verifierPermID uint64,
	agentPermID uint64,
	walletAgentPermID uint64,
) {
	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		panic(fmt.Sprintf("Failed to get creator address: %v", err))
	}

	// Create the session creation message
	msg := &permtypes.MsgCreateOrUpdatePermissionSession{
		Creator:           creatorAddr,
		Id:                sessionID,
		IssuerPermId:      issuerPermID,
		VerifierPermId:    verifierPermID,
		AgentPermId:       agentPermID,
		WalletAgentPermId: walletAgentPermID,
	}

	// Broadcast the transaction
	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		panic(fmt.Sprintf("Failed to create permission session: %v", err))
	}

	fmt.Print("CreatePermissionSession:\n\n")
	fmt.Println(txResp)
}

// VerifyPermissionSession verifies a permission session was created correctly with expected values
func VerifyPermissionSession(client cosmosclient.Client, ctx context.Context, sessionID string,
	expectedController string, expectedAgentPermID uint64, expectedIssuerPermID, expectedVerifierPermID uint64) bool {

	// Query the permission session
	permClient := permtypes.NewQueryClient(client.Context())
	resp, err := permClient.GetPermissionSession(ctx, &permtypes.QueryGetPermissionSessionRequest{
		Id: sessionID,
	})

	if err != nil {
		fmt.Printf("❌ Permission session verification failed: %v\n", err)
		return false
	}

	// Check if session exists
	if resp.GetSession() == nil {
		fmt.Printf("❌ Permission session verification failed: Session not found\n")
		return false
	}

	// Verify controller matches expectation if provided
	if expectedController != "" && resp.GetSession().Controller != expectedController {
		fmt.Printf("❌ Permission session verification failed: Expected controller %s, got %s\n",
			expectedController, resp.GetSession().Controller)
		return false
	}

	// Verify agent permission ID matches expectation
	if resp.GetSession().AgentPermId != expectedAgentPermID {
		fmt.Printf("❌ Permission session verification failed: Expected agent permission ID %d, got %d\n",
			expectedAgentPermID, resp.GetSession().AgentPermId)
		return false
	}

	// Check authorizations array for expected permissions
	issuerFound := false
	verifierFound := false
	for _, authz := range resp.GetSession().Authz {
		// Check for issuer permission ID if expected
		if expectedIssuerPermID > 0 && authz.GetExecutorPermId() == expectedIssuerPermID {
			issuerFound = true
		}

		// Check for verifier permission ID if expected
		if expectedVerifierPermID > 0 && authz.GetBeneficiaryPermId() == expectedVerifierPermID {
			verifierFound = true
		}
	}

	// Verify issuer permission if expected
	if expectedIssuerPermID > 0 && !issuerFound {
		fmt.Printf("❌ Permission session verification failed: Expected issuer permission ID %d not found in authorizations\n",
			expectedIssuerPermID)
		return false
	}

	// Verify verifier permission if expected
	if expectedVerifierPermID > 0 && !verifierFound {
		fmt.Printf("❌ Permission session verification failed: Expected verifier permission ID %d not found in authorizations\n",
			expectedVerifierPermID)
		return false
	}

	fmt.Printf("✅ Verified permission session ID %s with correct controller, agent permission ID %d",
		sessionID, expectedAgentPermID)

	if expectedIssuerPermID > 0 {
		fmt.Printf(", issuer permission ID %d", expectedIssuerPermID)
	}

	if expectedVerifierPermID > 0 {
		fmt.Printf(", verifier permission ID %d", expectedVerifierPermID)
	}

	fmt.Println()
	return true
}

// GetGovModuleAddress gets the governance module address
func GetGovModuleAddress(client cosmosclient.Client, ctx context.Context) (string, error) {
	// Use the deterministic module address from the SDK
	govModuleAddr := authtypes.NewModuleAddress("gov")
	return govModuleAddr.String(), nil
}

// GetYieldIntermediatePoolAddress gets the Yield Intermediate Pool module account address
func GetYieldIntermediatePoolAddress(client cosmosclient.Client, ctx context.Context) (string, error) {
	// The Yield Intermediate Pool is a module account with name "yield_intermediate_pool"
	// Derive it as the module account (this matches the blockchain code)
	yipModuleAddr := authtypes.NewModuleAddress("yield_intermediate_pool")
	return yipModuleAddr.String(), nil
}

// GetTrustDepositModuleAddress gets the Trust Deposit module account address
func GetTrustDepositModuleAddress(client cosmosclient.Client, ctx context.Context) (string, error) {
	// The Trust Deposit module account address
	tdModuleAddr := authtypes.NewModuleAddress("td")
	return tdModuleAddr.String(), nil
}

// GetProtocolPoolAddress gets the Protocol Pool (community pool) module account address
func GetProtocolPoolAddress(client cosmosclient.Client, ctx context.Context) (string, error) {
	protocolPoolAddr := authtypes.NewModuleAddress(protocolpooltypes.ModuleName)
	return protocolPoolAddr.String(), nil
}

// GetBlocksPerYear queries the mint module to get blocks_per_year
func GetBlocksPerYear(client cosmosclient.Client, ctx context.Context) (uint64, error) {
	mintClient := minttypes.NewQueryClient(client.Context())
	resp, err := mintClient.Params(ctx, &minttypes.QueryParamsRequest{})
	if err != nil {
		return 0, fmt.Errorf("failed to query mint parameters: %v", err)
	}
	return resp.Params.BlocksPerYear, nil
}

// GetDust gets the dust value from trust deposit module
// Note: Dust is not exposed via query endpoint, so we default to zero
// This matches the blockchain behavior where dust defaults to zero if not set
func GetDust(client cosmosclient.Client, ctx context.Context) (math.LegacyDec, error) {
	// Dust is stored internally in the keeper but not exposed via query
	// Default to zero as per blockchain code: dust defaults to zero if not set
	return math.LegacyZeroDec(), nil
}

// CalculateYieldAllowance calculates the yield allowance per block
// allowance = dust + trust_deposit_balance * trust_deposit_max_yield_rate / blocks_per_year
func CalculateYieldAllowance(
	trustDepositBalance math.Int,
	maxYieldRate math.LegacyDec,
	blocksPerYear uint64,
	dust math.LegacyDec,
) math.LegacyDec {
	trustDepositBalanceDec := math.LegacyNewDecFromInt(trustDepositBalance)
	blocksPerYearDec := math.LegacyNewDec(int64(blocksPerYear))
	perBlockYieldRate := maxYieldRate.Quo(blocksPerYearDec)
	perBlockYield := trustDepositBalanceDec.Mul(perBlockYieldRate)
	return dust.Add(perBlockYield)
}

// GetAllKnownTrustDeposits queries all known accounts and returns their trust deposits
func GetAllKnownTrustDeposits(client cosmosclient.Client, ctx context.Context) ([]struct {
	Address string
	Account cosmosaccount.Account
	TD      *tdtypes.TrustDeposit
}, error) {
	return GetAllKnownTrustDepositsAtHeight(client, ctx, 0)
}

// GetAllKnownTrustDepositsAtHeight queries all known accounts and returns their trust deposits at a specific block height
// If height is 0, queries at latest height
func GetAllKnownTrustDepositsAtHeight(client cosmosclient.Client, ctx context.Context, height int64) ([]struct {
	Address string
	Account cosmosaccount.Account
	TD      *tdtypes.TrustDeposit
}, error) {
	accounts := []struct {
		name    string
		address string
	}{
		{ISSUER_APPLICANT_NAME, ISSUER_APPLICANT_ADDRESS},
		{ISSUER_GRANTOR_APPLICANT_NAME, ISSUER_GRANTOR_APPLICANT_ADDRESS},
		{VERIFIER_APPLICANT_NAME, VERIFIER_APPLICANT_ADDRESS},
		{CREDENTIAL_HOLDER_NAME, CREDENTIAL_HOLDER_ADDRESS},
		{TRUST_REGISTRY_CONTROLLER_NAME, TRUST_REGISTRY_CONTROLLER_ADDRESS},
	}

	var results []struct {
		Address string
		Account cosmosaccount.Account
		TD      *tdtypes.TrustDeposit
	}

	for _, acc := range accounts {
		account, err := client.Account(acc.name)
		if err != nil {
			continue
		}

		td, err := GetTrustDepositAtHeight(client, ctx, account, height)
		if err != nil {
			continue // Account doesn't have a trust deposit
		}

		results = append(results, struct {
			Address string
			Account cosmosaccount.Account
			TD      *tdtypes.TrustDeposit
		}{
			Address: acc.address,
			Account: account,
			TD:      td,
		})
	}

	return results, nil
}

// SubmitSlashTrustDepositProposal submits a slash trust deposit governance proposal
func SubmitSlashTrustDepositProposal(
	client cosmosclient.Client,
	ctx context.Context,
	proposer cosmosaccount.Account,
	authority string,
	accountToSlash string,
	slashAmount uint64,
	title string,
	summary string,
) (uint64, error) {
	proposerAddr, err := proposer.Address(addressPrefix)
	if err != nil {
		return 0, fmt.Errorf("failed to get proposer address: %w", err)
	}

	// Create the slash trust deposit message
	slashMsg := &tdtypes.MsgSlashTrustDeposit{
		Authority: authority,
		Account:   accountToSlash,
		Amount:    math.NewIntFromUint64(slashAmount),
	}

	// Wrap in Any
	anyMsg, err := codectypes.NewAnyWithValue(slashMsg)
	if err != nil {
		return 0, fmt.Errorf("failed to create any message: %w", err)
	}

	// Parse deposit
	depositCoins, err := sdk.ParseCoinsNormalized("10000000uvna")
	if err != nil {
		return 0, fmt.Errorf("failed to parse deposit: %w", err)
	}

	// Create the submit proposal message
	msg := &govtypes.MsgSubmitProposal{
		Messages:       []*codectypes.Any{anyMsg},
		InitialDeposit: depositCoins,
		Proposer:       proposerAddr,
		Metadata:       "ipfs://CID",
		Title:          title,
		Summary:        summary,
		Expedited:      false,
	}

	txResp, err := client.BroadcastTx(ctx, proposer, msg)
	if err != nil {
		return 0, fmt.Errorf("failed to broadcast proposal: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("SubmitSlashTrustDepositProposal:\n\n")
	fmt.Println(txResp)

	// Extract proposal ID from events
	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal tx response: %w", err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal tx response: %w", err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "submit_proposal" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "proposal_id" {
					proposalID, err := strconv.ParseUint(attribute.Value, 10, 64)
					if err != nil {
						return 0, fmt.Errorf("failed to parse proposal ID: %w", err)
					}
					fmt.Printf("✅ Submitted governance proposal with ID: %d\n", proposalID)
					return proposalID, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("proposal ID not found in transaction response")
}

// VoteOnGovProposal votes on a governance proposal using gov v1
func VoteOnGovProposal(client cosmosclient.Client, ctx context.Context, voter cosmosaccount.Account, proposalID uint64, voteOption govtypes.VoteOption) error {
	voterAddr, err := voter.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get voter address: %w", err)
	}

	msg := &govtypes.MsgVote{
		ProposalId: proposalID,
		Voter:      voterAddr,
		Option:     voteOption,
		Metadata:   "",
	}

	txResp, err := client.BroadcastTx(ctx, voter, msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast vote: %w", err)
	}

	fmt.Print("VoteOnGovProposal:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("vote transaction failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	fmt.Printf("✅ Voted YES on proposal ID: %d\n", proposalID)
	return nil
}

// QueryGovProposal queries a governance proposal by ID
func QueryGovProposal(client cosmosclient.Client, ctx context.Context, proposalID uint64) (*govtypes.Proposal, error) {
	govClient := govtypes.NewQueryClient(client.Context())
	resp, err := govClient.Proposal(ctx, &govtypes.QueryProposalRequest{
		ProposalId: proposalID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query proposal: %w", err)
	}
	return resp.Proposal, nil
}

// SubmitContinuousFundProposal submits a governance proposal to create a continuous fund
func SubmitContinuousFundProposal(
	client cosmosclient.Client,
	ctx context.Context,
	proposer cosmosaccount.Account,
	authority string,
	recipient string,
	percentage string,
	title string,
	summary string,
) (uint64, error) {
	proposerAddr, err := proposer.Address(addressPrefix)
	if err != nil {
		return 0, fmt.Errorf("failed to get proposer address: %w", err)
	}

	// Parse percentage as decimal
	percentageDec, err := math.LegacyNewDecFromStr(percentage)
	if err != nil {
		return 0, fmt.Errorf("failed to parse percentage: %w", err)
	}

	// Create the MsgCreateContinuousFund message
	continuousFundMsg := &protocolpooltypes.MsgCreateContinuousFund{
		Authority:  authority,
		Recipient:  recipient,
		Percentage: percentageDec,
		Expiry:     nil, // No expiry
	}

	// Wrap in Any
	anyMsg, err := codectypes.NewAnyWithValue(continuousFundMsg)
	if err != nil {
		return 0, fmt.Errorf("failed to create any message: %w", err)
	}

	// Parse deposit
	depositCoins, err := sdk.ParseCoinsNormalized("10000000uvna")
	if err != nil {
		return 0, fmt.Errorf("failed to parse deposit: %w", err)
	}

	// Create the submit proposal message
	msg := &govtypes.MsgSubmitProposal{
		Messages:       []*codectypes.Any{anyMsg},
		InitialDeposit: depositCoins,
		Proposer:       proposerAddr,
		Metadata:       "ipfs://CID",
		Title:          title,
		Summary:        summary,
		Expedited:      false,
	}

	txResp, err := client.BroadcastTx(ctx, proposer, msg)
	if err != nil {
		return 0, fmt.Errorf("failed to broadcast proposal: %w", err)
	}

	// Print response from broadcasting a transaction
	fmt.Print("SubmitContinuousFundProposal:\n\n")
	fmt.Println(txResp)

	// Extract proposal ID from events
	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal tx response: %w", err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal tx response: %w", err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "submit_proposal" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "proposal_id" {
					proposalID, err := strconv.ParseUint(attribute.Value, 10, 64)
					if err != nil {
						return 0, fmt.Errorf("failed to parse proposal ID: %w", err)
					}
					fmt.Printf("✅ Submitted continuous fund governance proposal with ID: %d\n", proposalID)
					return proposalID, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("proposal ID not found in transaction response")
}

// WaitForProposalToPass waits for a proposal to pass by checking its status
func WaitForProposalToPass(client cosmosclient.Client, ctx context.Context, proposalID uint64, waitSeconds int) error {
	fmt.Printf("⏳ Waiting %d seconds for voting period to end...\n", waitSeconds)
	time.Sleep(time.Duration(waitSeconds) * time.Second)

	proposal, err := QueryGovProposal(client, ctx, proposalID)
	if err != nil {
		return fmt.Errorf("failed to query proposal: %w", err)
	}

	fmt.Printf("📊 Proposal status: %s\n", proposal.Status.String())

	if proposal.Status == govtypes.StatusPassed {
		fmt.Printf("✅ Proposal %d has PASSED\n", proposalID)
		return nil
	}

	return fmt.Errorf("proposal %d did not pass, status: %s", proposalID, proposal.Status.String())
}

// WaitForPermissionEffective polls until the permission becomes effective by checking block time.
// Uses polling with 1-second intervals and a configurable timeout (default 60s).
// This handles race conditions where block time may not have advanced sufficiently.
func WaitForPermissionEffective(client cosmosclient.Client, ctx context.Context, effectiveFrom time.Time, timeoutSeconds int) error {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 60 // Default 60s timeout as per PR #186 review
	}

	fmt.Printf("⏳ Polling for permission to become effective (timeout: %ds)...\n", timeoutSeconds)

	pollInterval := 1 * time.Second
	timeout := time.Duration(timeoutSeconds) * time.Second
	startTime := time.Now()

	for {
		// Get current block time
		blockTime, err := GetBlockTime(client, ctx)
		if err != nil {
			// If we can't get block time, wait and retry
			fmt.Printf("    ⚠️ Error getting block time: %v, retrying...\n", err)
			time.Sleep(pollInterval)
			continue
		}

		// Check if block time has passed effectiveFrom
		if blockTime.After(effectiveFrom) || blockTime.Equal(effectiveFrom) {
			elapsed := time.Since(startTime)
			fmt.Printf("    ✓ Permission is now effective (waited %.1fs, block time: %s)\n",
				elapsed.Seconds(), blockTime.Format(time.RFC3339))
			return nil
		}

		// Check timeout
		if time.Since(startTime) >= timeout {
			return fmt.Errorf("timeout after %ds waiting for permission to become effective. Block time: %s, effective_from: %s",
				timeoutSeconds, blockTime.Format(time.RFC3339), effectiveFrom.Format(time.RFC3339))
		}

		// Wait before next poll
		time.Sleep(pollInterval)
	}
}

// GetBlockTime gets the current block time from the blockchain
func GetBlockTime(client cosmosclient.Client, ctx context.Context) (time.Time, error) {
	// Query latest block via command line
	cmd := exec.Command("veranad", "q", "block", "-o", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query latest block: %v, output: %s", err, string(output))
	}

	// Remove any prefix text
	outputStr := string(output)
	jsonStart := -1
	for i := 0; i < len(outputStr); i++ {
		if outputStr[i] == '{' {
			jsonStart = i
			break
		}
	}
	if jsonStart == -1 {
		return time.Time{}, fmt.Errorf("no JSON found in output: %s", outputStr)
	}
	jsonOutput := outputStr[jsonStart:]

	var block struct {
		Header struct {
			Time string `json:"time"`
		} `json:"header"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &block); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse block JSON: %v, output: %s", err, jsonOutput)
	}

	blockTime, err := time.Parse(time.RFC3339Nano, block.Header.Time)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse block time: %v", err)
	}

	return blockTime, nil
}

// GetYIPIncomingAmountFromBlockResults queries block results for yield_distribution events
// Returns the YIP incoming balance from the yield_distribution event
// This queries via command line since cosmosclient doesn't expose block results directly
func GetYIPIncomingAmountFromBlockResults(blockHeight int64) (math.Int, error) {
	// Use exec to query block results via veranad command
	cmd := exec.Command("veranad", "q", "block-results", fmt.Sprintf("%d", blockHeight), "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return math.ZeroInt(), fmt.Errorf("failed to query block results: %v", err)
	}

	// Parse JSON to extract yield_distribution event
	var blockResults struct {
		FinalizeBlockEvents []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"finalize_block_events"`
	}

	if err := json.Unmarshal(output, &blockResults); err != nil {
		return math.ZeroInt(), fmt.Errorf("failed to parse block results JSON: %v", err)
	}

	// Find yield_distribution event and extract yip_incoming_balance
	for _, event := range blockResults.FinalizeBlockEvents {
		if event.Type == "yield_distribution" {
			for _, attr := range event.Attributes {
				if attr.Key == "yip_incoming_balance" {
					amount, ok := math.NewIntFromString(attr.Value)
					if !ok {
						return math.ZeroInt(), fmt.Errorf("failed to parse YIP incoming balance: %s", attr.Value)
					}
					return amount, nil
				}
			}
		}
	}

	return math.ZeroInt(), fmt.Errorf("yield_distribution event not found in block %d", blockHeight)
}

// GetBeginBlockTransferAmountFromBlockResults queries block results for yield_transfer events
// Returns the transfer amount from BeginBlock to TD module
func GetBeginBlockTransferAmountFromBlockResults(blockHeight int64) (math.Int, error) {
	// Use exec to query block results via veranad command
	cmd := exec.Command("veranad", "q", "block-results", fmt.Sprintf("%d", blockHeight), "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return math.ZeroInt(), fmt.Errorf("failed to query block results: %v", err)
	}

	// Parse JSON to extract yield_transfer event
	var blockResults struct {
		FinalizeBlockEvents []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"finalize_block_events"`
	}

	if err := json.Unmarshal(output, &blockResults); err != nil {
		return math.ZeroInt(), fmt.Errorf("failed to parse block results JSON: %v", err)
	}

	// Find yield_transfer event and extract transfer_amount
	for _, event := range blockResults.FinalizeBlockEvents {
		if event.Type == "yield_transfer" {
			for _, attr := range event.Attributes {
				if attr.Key == "transfer_amount" {
					amount, ok := math.NewIntFromString(attr.Value)
					if !ok {
						return math.ZeroInt(), fmt.Errorf("failed to parse transfer amount: %s", attr.Value)
					}
					return amount, nil
				}
			}
		}
	}

	// If no yield_transfer event found, BeginBlock didn't transfer anything (YIP was empty)
	return math.ZeroInt(), nil
}

// GetLatestBlockHeight gets the latest block height
func GetLatestBlockHeight(client cosmosclient.Client, ctx context.Context) (int64, error) {
	// Query latest block via command line
	cmd := exec.Command("veranad", "q", "block", "-o", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to query latest block: %v, output: %s", err, string(output))
	}

	// Remove any prefix text like "Falling back to latest block height:"
	outputStr := string(output)
	jsonStart := -1
	for i := 0; i < len(outputStr); i++ {
		if outputStr[i] == '{' {
			jsonStart = i
			break
		}
	}
	if jsonStart == -1 {
		return 0, fmt.Errorf("no JSON found in output: %s", outputStr)
	}
	jsonOutput := outputStr[jsonStart:]

	var block struct {
		Header struct {
			Height string `json:"height"`
		} `json:"header"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &block); err != nil {
		return 0, fmt.Errorf("failed to parse block JSON: %v, output: %s", err, jsonOutput)
	}

	height, err := strconv.ParseInt(block.Header.Height, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block height: %v", err)
	}

	return height, nil
}

// =============================================================================
// ERROR SCENARIO TESTING HELPERS
// =============================================================================
// These functions are designed for testing error scenarios.
// Unlike the standard helpers, they return errors instead of calling log.Fatal.

// ParseSchemaID parses a schema ID string to uint64
func ParseSchemaID(schemaIDStr string) uint64 {
	schemaID, err := strconv.ParseUint(schemaIDStr, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("failed to parse schema ID: %v", err))
	}
	return schemaID
}

// CreateRootPermissionWithError creates a root permission and returns any error
// instead of calling log.Fatal. This is useful for testing error scenarios.
func CreateRootPermissionWithError(client cosmosclient.Client, ctx context.Context,
	creator cosmosaccount.Account, msg permtypes.MsgCreateRootPermission) error {

	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get creator address: %v", err)
	}

	// Create the message
	fullMsg := &permtypes.MsgCreateRootPermission{
		Creator:          creatorAddr,
		SchemaId:         msg.SchemaId,
		Did:              msg.Did,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		ValidationFees:   msg.ValidationFees,
		VerificationFees: msg.VerificationFees,
		IssuanceFees:     msg.IssuanceFees,
		Country:          msg.Country,
	}

	txResp, err := client.BroadcastTx(ctx, creator, fullMsg)
	if err != nil {
		return fmt.Errorf("broadcast error: %v", err)
	}

	// Check transaction code - non-zero means error
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("transaction failed (code %d): %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return nil
}

// CreateRootPermissionAndGetID creates a root permission and returns its ID
// Returns the permission ID or an error
func CreateRootPermissionAndGetID(client cosmosclient.Client, ctx context.Context,
	creator cosmosaccount.Account, msg permtypes.MsgCreateRootPermission) (uint64, error) {

	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return 0, fmt.Errorf("failed to get creator address: %v", err)
	}

	// Create the message
	fullMsg := &permtypes.MsgCreateRootPermission{
		Creator:          creatorAddr,
		SchemaId:         msg.SchemaId,
		Did:              msg.Did,
		EffectiveFrom:    msg.EffectiveFrom,
		EffectiveUntil:   msg.EffectiveUntil,
		ValidationFees:   msg.ValidationFees,
		VerificationFees: msg.VerificationFees,
		IssuanceFees:     msg.IssuanceFees,
		Country:          msg.Country,
	}

	txResp, err := client.BroadcastTx(ctx, creator, fullMsg)
	if err != nil {
		return 0, fmt.Errorf("broadcast error: %v", err)
	}

	// Check transaction code
	if txResp.TxResponse.Code != 0 {
		return 0, fmt.Errorf("transaction failed (code %d): %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	// Extract permission ID from events
	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal tx response: %v", err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal tx response: %v", err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "create_root_permission" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "root_permission_id" {
					permID, parseErr := strconv.ParseUint(attribute.Value, 10, 64)
					if parseErr != nil {
						return 0, fmt.Errorf("failed to parse permission ID: %v", parseErr)
					}
					return permID, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("permission ID not found in events")
}

// StartPermissionVPWithError starts a permission VP and returns any error
// instead of calling log.Fatal. This is useful for testing error scenarios.
func StartPermissionVPWithError(client cosmosclient.Client, ctx context.Context,
	creator cosmosaccount.Account, msg permtypes.MsgStartPermissionVP) error {

	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get creator address: %v", err)
	}

	// Create the message
	fullMsg := &permtypes.MsgStartPermissionVP{
		Creator:          creatorAddr,
		Type:             msg.Type,
		Did:              msg.Did,
		ValidatorPermId:  msg.ValidatorPermId,
		Country:          msg.Country,
		ValidationFees:   msg.ValidationFees,
		IssuanceFees:     msg.IssuanceFees,
		VerificationFees: msg.VerificationFees,
	}

	txResp, err := client.BroadcastTx(ctx, creator, fullMsg)
	if err != nil {
		return fmt.Errorf("broadcast error: %v", err)
	}

	// Check transaction code - non-zero means error
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("transaction failed (code %d): %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return nil
}

// RevokePermissionWithError revokes a permission and returns any error
// instead of calling log.Fatal. This is useful for testing error scenarios.
func RevokePermissionWithError(client cosmosclient.Client, ctx context.Context,
	creator cosmosaccount.Account, msg permtypes.MsgRevokePermission) error {

	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get creator address: %v", err)
	}

	// Create the message
	fullMsg := &permtypes.MsgRevokePermission{
		Creator: creatorAddr,
		Id:      msg.Id,
	}

	txResp, err := client.BroadcastTx(ctx, creator, fullMsg)
	if err != nil {
		return fmt.Errorf("broadcast error: %v", err)
	}

	// Check transaction code - non-zero means error
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("transaction failed (code %d): %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return nil
}

// CreateInactiveValidatorPermission creates a permission with no effective_from
// This is an INACTIVE permission that can be used to test Issue #193
// Returns the permission ID or an error
func CreateInactiveValidatorPermission(client cosmosclient.Client, ctx context.Context,
	creator cosmosaccount.Account, schemaIDStr string, did string) (uint64, error) {

	// For ECOSYSTEM type permissions, we can create them directly without effective_from
	// But for this test, we need to create a permission programmatically that's inactive
	// Since the API now requires effective_from, we'll use a workaround:
	// Create a permission with future effective_from (making it INACTIVE/FUTURE state)

	schemaID := ParseSchemaID(schemaIDStr)
	futureTime := time.Now().Add(24 * time.Hour)
	farFuture := time.Now().Add(48 * time.Hour)

	creatorAddr, err := creator.Address(addressPrefix)
	if err != nil {
		return 0, fmt.Errorf("failed to get creator address: %v", err)
	}

	// Create an ECOSYSTEM (root) permission with future effective_from
	// This will be in FUTURE state (not ACTIVE)
	msg := &permtypes.MsgCreateRootPermission{
		Creator:        creatorAddr,
		SchemaId:       schemaID,
		Did:            did,
		EffectiveFrom:  &futureTime, // Future = not yet active
		EffectiveUntil: &farFuture,
	}

	txResp, err := client.BroadcastTx(ctx, creator, msg)
	if err != nil {
		return 0, fmt.Errorf("broadcast error: %v", err)
	}

	if txResp.TxResponse.Code != 0 {
		return 0, fmt.Errorf("transaction failed (code %d): %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	// Extract permission ID from events
	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal tx response: %v", err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal tx response: %v", err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "create_root_permission" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "root_permission_id" {
					permID, parseErr := strconv.ParseUint(attribute.Value, 10, 64)
					if parseErr != nil {
						return 0, fmt.Errorf("failed to parse permission ID: %v", parseErr)
					}
					fmt.Printf("   Created inactive (FUTURE) validator permission ID: %d\n", permID)
					return permID, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("permission ID not found in events")
}

// =============================================================================
// GROUP + OPERATOR AUTHORIZATION HELPERS
// =============================================================================

// CreateGroupWithPolicy creates a group with a threshold decision policy.
// Returns the group ID and group policy address.
func CreateGroupWithPolicy(
	client cosmosclient.Client,
	ctx context.Context,
	admin cosmosaccount.Account,
	memberAddresses []string,
	threshold string,
	votingPeriod time.Duration,
) (uint64, string, error) {
	adminAddr, err := admin.Address(addressPrefix)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get admin address: %w", err)
	}

	// Build member requests
	members := make([]group.MemberRequest, len(memberAddresses))
	for i, addr := range memberAddresses {
		members[i] = group.MemberRequest{
			Address:  addr,
			Weight:   "1",
			Metadata: fmt.Sprintf("member_%d", i+1),
		}
	}

	// Create threshold decision policy
	decisionPolicy := &group.ThresholdDecisionPolicy{
		Threshold: threshold,
		Windows: &group.DecisionPolicyWindows{
			VotingPeriod: votingPeriod,
		},
	}
	decisionPolicyAny, err := codectypes.NewAnyWithValue(decisionPolicy)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create decision policy any: %w", err)
	}

	msg := &group.MsgCreateGroupWithPolicy{
		Admin:               adminAddr,
		Members:             members,
		GroupMetadata:       "test group for TR authz",
		GroupPolicyMetadata: "threshold policy",
		GroupPolicyAsAdmin:  true,
		DecisionPolicy:     decisionPolicyAny,
	}

	txResp, err := client.BroadcastTx(ctx, admin, msg)
	if err != nil {
		return 0, "", fmt.Errorf("failed to broadcast CreateGroupWithPolicy: %w", err)
	}

	fmt.Print("CreateGroupWithPolicy:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return 0, "", fmt.Errorf("CreateGroupWithPolicy failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	// Extract group ID and policy address from events
	var groupID uint64
	var policyAddr string

	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal tx response: %w", err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		return 0, "", fmt.Errorf("failed to unmarshal tx response: %w", err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "cosmos.group.v1.EventCreateGroup" {
			for _, attr := range event.Attributes {
				if attr.Key == "group_id" {
					groupID, _ = strconv.ParseUint(strings.Trim(attr.Value, "\""), 10, 64)
				}
			}
		}
		if event.Type == "cosmos.group.v1.EventCreateGroupPolicy" {
			for _, attr := range event.Attributes {
				if attr.Key == "address" {
					policyAddr = strings.Trim(attr.Value, "\"")
				}
			}
		}
	}

	if groupID == 0 || policyAddr == "" {
		return 0, "", fmt.Errorf("failed to extract group ID or policy address from events")
	}

	fmt.Printf("✅ Created group ID: %d, policy address: %s\n", groupID, policyAddr)
	return groupID, policyAddr, nil
}

// SubmitGroupProposal submits a group proposal with the given messages.
// Returns the proposal ID.
func SubmitGroupProposal(
	client cosmosclient.Client,
	ctx context.Context,
	proposer cosmosaccount.Account,
	policyAddr string,
	innerMsgs []sdk.Msg,
	title string,
	summary string,
) (uint64, error) {
	proposerAddr, err := proposer.Address(addressPrefix)
	if err != nil {
		return 0, fmt.Errorf("failed to get proposer address: %w", err)
	}

	// Wrap messages in Any
	anyMsgs := make([]*codectypes.Any, len(innerMsgs))
	for i, innerMsg := range innerMsgs {
		anyMsg, err := codectypes.NewAnyWithValue(innerMsg)
		if err != nil {
			return 0, fmt.Errorf("failed to create any for msg %d: %w", i, err)
		}
		anyMsgs[i] = anyMsg
	}

	msg := &group.MsgSubmitProposal{
		GroupPolicyAddress: policyAddr,
		Proposers:          []string{proposerAddr},
		Messages:           anyMsgs,
		Exec:               group.Exec_EXEC_UNSPECIFIED,
		Title:              title,
		Summary:            summary,
	}

	txResp, err := client.BroadcastTx(ctx, proposer, msg)
	if err != nil {
		return 0, fmt.Errorf("failed to broadcast SubmitGroupProposal: %w", err)
	}

	fmt.Print("SubmitGroupProposal:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return 0, fmt.Errorf("SubmitGroupProposal failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	// Extract proposal ID from events
	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal tx response: %w", err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal tx response: %w", err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "cosmos.group.v1.EventSubmitProposal" {
			for _, attr := range event.Attributes {
				if attr.Key == "proposal_id" {
					proposalID, parseErr := strconv.ParseUint(strings.Trim(attr.Value, "\""), 10, 64)
					if parseErr != nil {
						return 0, fmt.Errorf("failed to parse proposal ID: %w", parseErr)
					}
					fmt.Printf("✅ Submitted group proposal ID: %d\n", proposalID)
					return proposalID, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("proposal ID not found in events")
}

// VoteOnGroupProposal votes YES on a group proposal.
// If tryExec is true, uses EXEC_TRY for auto-execution.
func VoteOnGroupProposal(
	client cosmosclient.Client,
	ctx context.Context,
	voter cosmosaccount.Account,
	proposalID uint64,
	tryExec bool,
) error {
	voterAddr, err := voter.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get voter address: %w", err)
	}

	execMode := group.Exec_EXEC_UNSPECIFIED
	if tryExec {
		execMode = group.Exec_EXEC_TRY
	}

	msg := &group.MsgVote{
		ProposalId: proposalID,
		Voter:      voterAddr,
		Option:     group.VOTE_OPTION_YES,
		Exec:       execMode,
	}

	txResp, err := client.BroadcastTx(ctx, voter, msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast VoteOnGroupProposal: %w", err)
	}

	fmt.Print("VoteOnGroupProposal:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("VoteOnGroupProposal failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	fmt.Printf("✅ Voted YES on group proposal %d (tryExec=%v)\n", proposalID, tryExec)
	return nil
}

// ExecGroupProposal executes a group proposal.
func ExecGroupProposal(
	client cosmosclient.Client,
	ctx context.Context,
	executor cosmosaccount.Account,
	proposalID uint64,
) error {
	executorAddr, err := executor.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get executor address: %w", err)
	}

	msg := &group.MsgExec{
		ProposalId: proposalID,
		Executor:   executorAddr,
	}

	txResp, err := client.BroadcastTx(ctx, executor, msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast ExecGroupProposal: %w", err)
	}

	fmt.Print("ExecGroupProposal:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("ExecGroupProposal failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	fmt.Printf("✅ Executed group proposal %d\n", proposalID)
	return nil
}

// GrantOperatorAuthorizationViaGroup grants operator authorization via a group proposal.
// Submits a proposal containing MsgGrantOperatorAuthorization, votes from admin and voter2, auto-executes.
func GrantOperatorAuthorizationViaGroup(
	client cosmosclient.Client,
	ctx context.Context,
	admin cosmosaccount.Account,
	voter2 cosmosaccount.Account,
	policyAddr string,
	operatorAddr string,
	granteeAddr string,
	msgTypes []string,
) error {
	// Create the MsgGrantOperatorAuthorization
	// Operator is empty because this is executed via group proposal (authority signs directly)
	grantMsg := &detypes.MsgGrantOperatorAuthorization{
		Authority: policyAddr,
		Operator:  "",
		Grantee:   granteeAddr,
		MsgTypes:  msgTypes,
	}

	// Submit group proposal
	proposalID, err := SubmitGroupProposal(
		client, ctx, admin, policyAddr,
		[]sdk.Msg{grantMsg},
		fmt.Sprintf("Grant operator auth for %v", msgTypes),
		"Grant operator authorization via group proposal",
	)
	if err != nil {
		return fmt.Errorf("failed to submit group proposal: %w", err)
	}

	// Wait for proposal tx to be fully processed before admin votes
	fmt.Println("    - Waiting for proposal to be processed...")
	time.Sleep(3 * time.Second)

	// Vote YES from admin
	err = VoteOnGroupProposal(client, ctx, admin, proposalID, false)
	if err != nil {
		return fmt.Errorf("failed to vote from admin: %w", err)
	}

	// Wait for admin vote to be processed before voter2 votes
	fmt.Println("    - Waiting for admin vote to be processed...")
	time.Sleep(3 * time.Second)

	// Vote YES from voter2 with EXEC_TRY (auto-execute on threshold met)
	err = VoteOnGroupProposal(client, ctx, voter2, proposalID, true)
	if err != nil {
		return fmt.Errorf("failed to vote from voter2: %w", err)
	}

	// Wait for execution to complete
	fmt.Println("    - Waiting for proposal execution to complete...")
	time.Sleep(3 * time.Second)

	fmt.Printf("✅ Granted operator authorization for %s via group proposal\n", granteeAddr)
	return nil
}

// CreateTrustRegistryWithAuthority creates a trust registry where authority and operator are different.
func CreateTrustRegistryWithAuthority(
	client cosmosclient.Client,
	ctx context.Context,
	operatorAccount cosmosaccount.Account,
	authority string,
	did string,
	aka string,
	docURL string,
	docHash string,
	language string,
) (string, error) {
	operatorAddr, err := operatorAccount.Address(addressPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to get operator address: %w", err)
	}

	msg := &trtypes.MsgCreateTrustRegistry{
		Authority:    authority,
		Operator:     operatorAddr,
		Did:          did,
		Aka:          aka,
		Language:     language,
		DocUrl:       docURL,
		DocDigestSri: docHash,
	}

	txResp, err := client.BroadcastTx(ctx, operatorAccount, msg)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast CreateTrustRegistry: %w", err)
	}

	fmt.Print("CreateTrustRegistryWithAuthority:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("CreateTrustRegistry failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	var txResponse sdk.TxResponse
	txResponseBytes, err := client.Context().Codec.MarshalJSON(txResp.TxResponse)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tx response: %w", err)
	}
	err = client.Context().Codec.UnmarshalJSON(txResponseBytes, &txResponse)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal tx response: %w", err)
	}

	for _, event := range txResponse.Events {
		if event.Type == "create_trust_registry" {
			for _, attr := range event.Attributes {
				if attr.Key == "trust_registry_id" {
					return attr.Value, nil
				}
			}
		}
	}
	return "", fmt.Errorf("trust_registry_id not found in events")
}

// AddGFDWithAuthority adds a governance framework document with separate authority/operator.
func AddGFDWithAuthority(
	client cosmosclient.Client,
	ctx context.Context,
	operatorAccount cosmosaccount.Account,
	authority string,
	trID uint64,
	docLanguage string,
	docURL string,
	docHash string,
	version int32,
) error {
	operatorAddr, err := operatorAccount.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get operator address: %w", err)
	}

	msg := &trtypes.MsgAddGovernanceFrameworkDocument{
		Authority:    authority,
		Operator:     operatorAddr,
		Id:           trID,
		DocLanguage:  docLanguage,
		DocUrl:       docURL,
		DocDigestSri: docHash,
		Version:      version,
	}

	txResp, err := client.BroadcastTx(ctx, operatorAccount, msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast AddGFD: %w", err)
	}

	fmt.Print("AddGFDWithAuthority:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("AddGFD failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return nil
}

// IncreaseActiveGFVersionWithAuthority increases the active GF version with separate authority/operator.
func IncreaseActiveGFVersionWithAuthority(
	client cosmosclient.Client,
	ctx context.Context,
	operatorAccount cosmosaccount.Account,
	authority string,
	trID uint64,
) error {
	operatorAddr, err := operatorAccount.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get operator address: %w", err)
	}

	msg := &trtypes.MsgIncreaseActiveGovernanceFrameworkVersion{
		Authority: authority,
		Operator:  operatorAddr,
		Id:        trID,
	}

	txResp, err := client.BroadcastTx(ctx, operatorAccount, msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast IncreaseActiveGFVersion: %w", err)
	}

	fmt.Print("IncreaseActiveGFVersionWithAuthority:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("IncreaseActiveGFVersion failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return nil
}

// UpdateTrustRegistryWithAuthority updates a trust registry with separate authority/operator.
func UpdateTrustRegistryWithAuthority(
	client cosmosclient.Client,
	ctx context.Context,
	operatorAccount cosmosaccount.Account,
	authority string,
	trID uint64,
	did string,
	aka string,
) error {
	operatorAddr, err := operatorAccount.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get operator address: %w", err)
	}

	msg := &trtypes.MsgUpdateTrustRegistry{
		Authority: authority,
		Operator:  operatorAddr,
		Id:        trID,
		Did:       did,
		Aka:       aka,
	}

	txResp, err := client.BroadcastTx(ctx, operatorAccount, msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast UpdateTrustRegistry: %w", err)
	}

	fmt.Print("UpdateTrustRegistryWithAuthority:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("UpdateTrustRegistry failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return nil
}

// ArchiveTrustRegistryWithAuthority archives/unarchives a trust registry with separate authority/operator.
func ArchiveTrustRegistryWithAuthority(
	client cosmosclient.Client,
	ctx context.Context,
	operatorAccount cosmosaccount.Account,
	authority string,
	trID uint64,
	archive bool,
) error {
	operatorAddr, err := operatorAccount.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("failed to get operator address: %w", err)
	}

	msg := &trtypes.MsgArchiveTrustRegistry{
		Authority: authority,
		Operator:  operatorAddr,
		Id:        trID,
		Archive:   archive,
	}

	txResp, err := client.BroadcastTx(ctx, operatorAccount, msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast ArchiveTrustRegistry: %w", err)
	}

	fmt.Print("ArchiveTrustRegistryWithAuthority:\n\n")
	fmt.Println(txResp)

	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("ArchiveTrustRegistry failed with code %d: %s",
			txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}

	return nil
}
