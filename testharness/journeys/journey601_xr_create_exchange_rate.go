package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cstypes "github.com/verana-labs/verana/x/cs/types"
	xrtypes "github.com/verana-labs/verana/x/xr/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// submitXrGovProposal submits a governance proposal containing an XR message,
// votes YES from cooluser (the validator), waits for it to pass, and returns the proposal ID.
func submitXrGovProposal(
	client cosmosclient.Client,
	ctx context.Context,
	proposer string,
	proposerAccount interface{ Address(string) (string, error) },
	msg sdk.Msg,
	title string,
	summary string,
) (uint64, error) {
	anyMsg, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to create any message: %w", err)
	}

	depositCoins, err := sdk.ParseCoinsNormalized("10000000uvna")
	if err != nil {
		return 0, fmt.Errorf("failed to parse deposit: %w", err)
	}

	submitMsg := &govtypes.MsgSubmitProposal{
		Messages:       []*codectypes.Any{anyMsg},
		InitialDeposit: depositCoins,
		Proposer:       proposer,
		Metadata:       "ipfs://CID",
		Title:          title,
		Summary:        summary,
		Expedited:      false,
	}

	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)
	txResp, err := client.BroadcastTx(ctx, cooluser, submitMsg)
	if err != nil {
		return 0, fmt.Errorf("failed to broadcast proposal: %w", err)
	}

	if txResp.TxResponse.Code != 0 {
		return 0, fmt.Errorf("proposal submission failed with code %d: %s",
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

// voteAndPassGovProposal votes YES on a proposal and waits for it to pass.
func voteAndPassGovProposal(
	client cosmosclient.Client,
	ctx context.Context,
	proposalID uint64,
) error {
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	fmt.Println("    - Waiting for proposal to be processed...")
	time.Sleep(3 * time.Second)

	err := lib.VoteOnGovProposal(client, ctx, cooluser, proposalID, govtypes.OptionYes)
	if err != nil {
		return fmt.Errorf("failed to vote on proposal %d: %w", proposalID, err)
	}

	// Poll for the proposal to leave the voting period and pass. The local chain
	// uses a 30s voting period, so poll well past that before giving up.
	fmt.Println("    - Waiting for voting period to end...")
	var proposal *govtypes.Proposal
	for i := 0; i < 30; i++ {
		time.Sleep(3 * time.Second)
		proposal, err = lib.QueryGovProposal(client, ctx, proposalID)
		if err != nil {
			return fmt.Errorf("failed to query proposal %d: %w", proposalID, err)
		}
		if proposal.Status != govtypes.StatusVotingPeriod && proposal.Status != govtypes.StatusDepositPeriod {
			break
		}
	}

	fmt.Printf("    Proposal status: %s\n", proposal.Status.String())
	if proposal.Status != govtypes.StatusPassed {
		return fmt.Errorf("proposal %d did not pass, status: %s", proposalID, proposal.Status.String())
	}

	fmt.Printf("✅ Proposal %d has PASSED\n", proposalID)
	return nil
}

// RunXrCreateExchangeRateJourney implements Journey 601: XR Create Exchange Rate (governance)
// Creates an exchange rate via governance proposal, toggles state to true.
func RunXrCreateExchangeRateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 601: XR Create Exchange Rate via Governance")

	govModuleAddr := authtypes.NewModuleAddress("gov").String()
	coolusrAddr := lib.COOLUSER_ADDRESS
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	fmt.Printf("  Gov Module:  %s\n", govModuleAddr)
	fmt.Printf("  Proposer:    %s\n", coolusrAddr)

	// =========================================================================
	// Step 1: Submit governance proposal to create an exchange rate
	// =========================================================================
	fmt.Println("\n--- Step 1: Submit CreateExchangeRate governance proposal ---")

	createMsg := &xrtypes.MsgCreateExchangeRate{
		Authority:        govModuleAddr,
		BaseAssetType:    cstypes.PricingAssetType_TU,
		BaseAsset:        "TU",
		QuoteAssetType:   cstypes.PricingAssetType_COIN,
		QuoteAsset:       "uvna",
		Rate:             "1000000",
		RateScale:        6,
		ValidityDuration: 24 * time.Hour,
	}

	proposalID, err := submitXrGovProposal(
		client, ctx, coolusrAddr, cooluser,
		createMsg,
		"Create TU/uvna Exchange Rate",
		"Create exchange rate for TU to uvna conversion with rate=1000000, scale=6, validity=24h",
	)
	if err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	fmt.Printf("✅ Step 1: Submitted CreateExchangeRate proposal (ID: %d)\n", proposalID)

	// =========================================================================
	// Step 2: Vote and pass the proposal
	// =========================================================================
	fmt.Println("\n--- Step 2: Vote and pass the proposal ---")

	err = voteAndPassGovProposal(client, ctx, proposalID)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Println("✅ Step 2: CreateExchangeRate proposal passed")

	// =========================================================================
	// Step 3: Query the exchange rate to verify it was created with state=false
	// =========================================================================
	fmt.Println("\n--- Step 3: Query exchange rate to verify creation ---")

	xrQueryClient := xrtypes.NewQueryClient(client.Context())
	listResp, err := xrQueryClient.ListExchangeRates(ctx, &xrtypes.QueryListExchangeRatesRequest{
		BaseAssetType:  cstypes.PricingAssetType_TU,
		BaseAsset:      "TU",
		QuoteAssetType: cstypes.PricingAssetType_COIN,
		QuoteAsset:     "uvna",
	})
	if err != nil {
		return fmt.Errorf("step 3 failed: could not list exchange rates: %w", err)
	}

	if len(listResp.ExchangeRates) == 0 {
		return fmt.Errorf("step 3 failed: no exchange rates found for TU/uvna")
	}

	// Use the last created exchange rate
	xr := listResp.ExchangeRates[len(listResp.ExchangeRates)-1]
	exchangeRateID := xr.Id

	fmt.Printf("  Exchange Rate ID: %d\n", exchangeRateID)
	fmt.Printf("  Base:  %s (%s)\n", xr.BaseAsset, xr.BaseAssetType)
	fmt.Printf("  Quote: %s (%s)\n", xr.QuoteAsset, xr.QuoteAssetType)
	fmt.Printf("  Rate:  %s (scale: %d)\n", xr.Rate, xr.RateScale)
	fmt.Printf("  State: %v\n", xr.State)

	// CreateExchangeRate activates the rate on creation (state=true).
	if xr.State != true {
		return fmt.Errorf("step 3 failed: expected state=true, got state=%v", xr.State)
	}
	fmt.Println("✅ Step 3: Exchange rate created with state=true")

	// =========================================================================
	// Step 4-6: Exercise SetExchangeRateState. Per [MOD-XR-MSG-3] the handler
	// TOGGLES the stored state (it ignores any explicit value). The rate is
	// created active (true), so the first toggle disables it (false) and the
	// second re-enables it (true), leaving it active for downstream journeys.
	// =========================================================================
	toggle := func(label string, expected bool) error {
		fmt.Printf("\n--- %s: SetExchangeRateState (toggle) -> expect state=%v ---\n", label, expected)
		pid, err := submitXrGovProposal(
			client, ctx, coolusrAddr, cooluser,
			&xrtypes.MsgSetExchangeRateState{Authority: govModuleAddr, Id: exchangeRateID},
			"Toggle TU/uvna Exchange Rate State", "Toggle exchange rate state",
		)
		if err != nil {
			return err
		}
		if err := voteAndPassGovProposal(client, ctx, pid); err != nil {
			return err
		}
		resp, err := xrQueryClient.GetExchangeRate(ctx, &xrtypes.QueryGetExchangeRateRequest{Id: exchangeRateID})
		if err != nil {
			return fmt.Errorf("could not query exchange rate: %w", err)
		}
		if resp.ExchangeRate.State != expected {
			return fmt.Errorf("expected state=%v, got state=%v", expected, resp.ExchangeRate.State)
		}
		fmt.Printf("✅ %s: state is now %v\n", label, resp.ExchangeRate.State)
		return nil
	}

	if err := toggle("Step 4-5", false); err != nil {
		return fmt.Errorf("step 4-5 failed: %w", err)
	}
	if err := toggle("Step 6", true); err != nil {
		return fmt.Errorf("step 6 failed: %w", err)
	}

	// =========================================================================
	// Step 7: Save exchange rate ID for downstream journeys
	// =========================================================================
	result := lib.JourneyResult{
		ExchangeRateID: strconv.FormatUint(exchangeRateID, 10),
	}
	lib.SaveJourneyResult("journey601", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 601 completed successfully!")
	fmt.Println("XR CreateExchangeRate via Governance tested:")
	fmt.Println("  - CreateExchangeRate: proposal submitted and passed (state=true)")
	fmt.Println("  - SetExchangeRateState: toggled true->false then false->true")
	fmt.Println("  - Exchange rate left active for downstream journeys")
	fmt.Printf("  - Exchange Rate ID: %d\n", exchangeRateID)
	fmt.Println("========================================")

	return nil
}
