package de

import (
	"fmt"
	"strconv"

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
	cmd.AddCommand(CmdListVSOperatorAuthorizations())
	cmd.AddCommand(CmdGetOperatorAuthorization())
	cmd.AddCommand(CmdGetVSOperatorAuthorization())

	return cmd
}

// CmdGetOperatorAuthorization returns a cobra command for the
// [MOD-DE-QRY-3] GetOperatorAuthorization query.
func CmdGetOperatorAuthorization() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-operator-authorization [id]",
		Short: "Get an operator authorization by id",
		Long:  "[MOD-DE-QRY-3] Get a single operator authorization by its id.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.GetOperatorAuthorization(cmd.Context(), &types.QueryGetOperatorAuthorizationRequest{Id: id})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdGetVSOperatorAuthorization returns a cobra command for the
// [MOD-DE-QRY-4] GetVSOperatorAuthorization query.
func CmdGetVSOperatorAuthorization() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-vs-operator-authorization [id]",
		Short: "Get a VS operator authorization by id",
		Long:  "[MOD-DE-QRY-4] Get a single VS operator authorization (with its records) by its id.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.GetVSOperatorAuthorization(cmd.Context(), &types.QueryGetVSOperatorAuthorizationRequest{Id: id})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdListOperatorAuthorizations returns a cobra command for the
// [MOD-DE-QRY-1] ListOperatorAuthorizations query.
func CmdListOperatorAuthorizations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-operator-authorizations",
		Short: "List operator authorizations with optional filters",
		Long:  "[MOD-DE-QRY-1] List operator authorizations. Optionally filter by corporation and/or operator address.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			corporationID, _ := cmd.Flags().GetUint64("corporation-id")
			operator, _ := cmd.Flags().GetString("operator")
			limit, _ := cmd.Flags().GetUint32("limit")

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.ListOperatorAuthorizations(cmd.Context(), &types.QueryListOperatorAuthorizationsRequest{
				CorporationId:   corporationID,
				Operator:        operator,
				ResponseMaxSize: limit,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint64("corporation-id", 0, "filter by the corporation id that granted the authorization")
	cmd.Flags().String("operator", "", "filter by the operator account that received the authorization")
	cmd.Flags().Uint32("limit", 64, "maximum number of results (1-1024, default 64)")

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// CmdListVSOperatorAuthorizations returns a cobra command for the
// [MOD-DE-QRY-2] ListVSOperatorAuthorizations query.
func CmdListVSOperatorAuthorizations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-vs-operator-authorizations",
		Short: "List VS operator authorizations with optional filters",
		Long:  "[MOD-DE-QRY-2] List VS operator authorizations. Optionally filter by corporation and/or vs_operator address.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			corporationID, _ := cmd.Flags().GetUint64("corporation-id")
			vsOperator, _ := cmd.Flags().GetString("vs-operator")
			limit, _ := cmd.Flags().GetUint32("limit")

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.ListVSOperatorAuthorizations(cmd.Context(), &types.QueryListVSOperatorAuthorizationsRequest{
				CorporationId:   corporationID,
				VsOperator:      vsOperator,
				ResponseMaxSize: limit,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint64("corporation-id", 0, "filter by the corporation id that granted the authorization")
	cmd.Flags().String("vs-operator", "", "filter by the VS operator account")
	cmd.Flags().Uint32("limit", 64, "maximum number of results (1-1024, default 64)")

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
