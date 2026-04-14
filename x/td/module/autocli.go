package trustdeposit

import (
	"fmt"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/spf13/cobra"
	"github.com/verana-labs/verana/x/td/types"

	modulev1 "github.com/verana-labs/verana/api/verana/td/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "GetTrustDeposit",
					Use:       "get-trust-deposit [corporation]",
					Short:     "Query trust deposit for a corporation",
					Long:      "Get the trust deposit information for a given corporation address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{
							ProtoField: "corporation",
						},
					},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "ReclaimTrustDepositYield",
					Use:       "reclaim-yield [corporation]",
					Short:     "Reclaim earned interest from trust deposits",
					Long:      "Reclaim any available interest earned from trust deposits. The interest is calculated based on share value and current deposit amount.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
					},
				},
				{
					RpcMethod: "RepaySlashedTrustDeposit",
					Use:       "repay-slashed-td [corporation] [deposit]",
					Short:     "Repay slashed trust deposit",
					Long:      "Repay the outstanding slashed trust deposit. The deposit must exactly match the outstanding slashed amount.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "corporation"},
						{ProtoField: "deposit"},
					},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}

func CmdSlashTrustDepositProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slash-trust-deposit [corporation] [deposit] [flags]",
		Short: "Submit a governance proposal to slash a corporation's trust deposit",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Parse corporation address
			corporation := args[0]
			if _, err := sdk.AccAddressFromBech32(corporation); err != nil {
				return err
			}

			// Parse slash deposit amount
			slashDeposit, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid deposit: %s", args[1])
			}

			// Get proposal details from flags
			title, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			// Create the proposal content
			content := types.NewSlashTrustDepositProposal(title, description, corporation, slashDeposit)

			// Create the governance proposal message
			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "The proposal title")
	cmd.Flags().String(cli.FlagDescription, "", "The proposal description")
	cmd.Flags().String(cli.FlagDeposit, "", "The proposal deposit")
	err := cmd.MarkFlagRequired(cli.FlagTitle)
	if err != nil {
		return nil
	}
	err = cmd.MarkFlagRequired(cli.FlagDescription)
	if err != nil {
		return nil
	}
	err = cmd.MarkFlagRequired(cli.FlagDeposit)
	if err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

var SlashTrustDepositHandler = govclient.NewProposalHandler(CmdSlashTrustDepositProposal)
