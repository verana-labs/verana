package de

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/verana-labs/verana/x/de/types"
)

// GetQueryCmd implements the autocli.HasCustomQueryCommand interface.
// This is needed because autocli's amino JSON encoder cannot properly render
// gogo proto types with extensions (stdtime, stdduration, castrepeated) used
// in OperatorAuthorization. The custom command uses the gogo proto codec directly.
func (am AppModule) GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   types.ModuleName,
		Short: fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
	}

	cmd.AddCommand(CmdListOperatorAuthorizations())

	return cmd
}

// CmdListOperatorAuthorizations returns a cobra command for the
// [MOD-DE-QRY-1] ListOperatorAuthorizations query.
func CmdListOperatorAuthorizations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-operator-authorizations",
		Short: "List operator authorizations with optional filters",
		Long:  "[MOD-DE-QRY-1] List operator authorizations. Optionally filter by authority and/or operator address.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			authority, _ := cmd.Flags().GetString("authority")
			operator, _ := cmd.Flags().GetString("operator")
			limit, _ := cmd.Flags().GetUint32("limit")

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.ListOperatorAuthorizations(cmd.Context(), &types.QueryListOperatorAuthorizationsRequest{
				Authority:       authority,
				Operator:        operator,
				ResponseMaxSize: limit,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().String("authority", "", "filter by the authority group that granted the authorization")
	cmd.Flags().String("operator", "", "filter by the operator account that received the authorization")
	cmd.Flags().Uint32("limit", 64, "maximum number of results (1-1024, default 64)")

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
