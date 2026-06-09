package journeys

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cstypes "github.com/verana-labs/verana/x/cs/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunCsSchemaAuthPolicyJourney implements Journey 311: CS Schema Authorization Policy.
// Exercises MOD-CS-MSG-5/6/7: Create -> IncreaseActiveVersion -> Revoke a schema
// authorization policy, operator-signed under group-granted operator authorization.
// Depends on Journey 301/302 (corp + operator) and Journey 307 (credential schema).
func RunCsSchemaAuthPolicyJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 311: CS Schema Authorization Policy (create/increase/revoke)")

	setup301 := lib.LoadJourneyResult("journey301")
	setup307 := lib.LoadJourneyResult("journey307")

	policyAddr := setup301.GroupPolicyAddr
	operatorAddr := setup301.OperatorAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	schemaID, err := strconv.ParseUint(setup307.SchemaID, 10, 64)
	if err != nil {
		return fmt.Errorf("step 0 failed: could not parse schema id from journey307: %w", err)
	}
	fmt.Printf("  Corporation: %s\n  Operator: %s\n  Schema ID: %d\n", policyAddr, operatorAddr, schemaID)

	// Step 1: Grant operator authz for the 3 CS policy msg types
	fmt.Println("\n--- Step 1: Grant operator authz for CS schema-auth-policy msgs ---")
	err = lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr,
		[]string{
			"/verana.cs.v1.MsgCreateSchemaAuthorizationPolicy",
			"/verana.cs.v1.MsgIncreaseActiveSchemaAuthorizationPolicyVersion",
			"/verana.cs.v1.MsgRevokeSchemaAuthorizationPolicy",
		},
	)
	if err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	fmt.Println("OK Step 1: operator authorized for CS policy msgs")
	waitForTx("grant cs policy authz")

	role := cstypes.SchemaAuthorizationPolicyRole_SCHEMA_AUTHORIZATION_POLICY_ROLE_ISSUER

	// Step 2: Create schema authorization policy (MOD-CS-MSG-5)
	fmt.Println("\n--- Step 2: CreateSchemaAuthorizationPolicy (ISSUER) ---")
	createMsg := &cstypes.MsgCreateSchemaAuthorizationPolicy{
		Corporation: policyAddr,
		Operator:    operatorAddr,
		SchemaId:    schemaID,
		Role:        role,
		Url:         "https://example.com/issuer-policy.json",
		DigestSri:   "sha384-AbCdEf0123456789AbCdEf0123456789AbCdEf0123456789AbCdEf0123456789",
	}
	resp, err := client.BroadcastTx(ctx, operatorAccount, createMsg)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	if resp.TxResponse.Code != 0 {
		return fmt.Errorf("step 2 failed code %d: %s", resp.TxResponse.Code, resp.TxResponse.RawLog)
	}
	fmt.Println("OK Step 2: schema authorization policy created (version 1)")
	waitForTx("create cs policy")

	// Step 3: Increase active version (MOD-CS-MSG-6)
	fmt.Println("\n--- Step 3: IncreaseActiveSchemaAuthorizationPolicyVersion ---")
	incMsg := &cstypes.MsgIncreaseActiveSchemaAuthorizationPolicyVersion{
		Corporation: policyAddr,
		Operator:    operatorAddr,
		SchemaId:    schemaID,
		Role:        role,
	}
	resp, err = client.BroadcastTx(ctx, operatorAccount, incMsg)
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	if resp.TxResponse.Code != 0 {
		return fmt.Errorf("step 3 failed code %d: %s", resp.TxResponse.Code, resp.TxResponse.RawLog)
	}
	fmt.Println("OK Step 3: active policy version increased")
	waitForTx("increase cs policy version")

	// Step 4: Revoke schema authorization policy version 1 (MOD-CS-MSG-7)
	fmt.Println("\n--- Step 4: RevokeSchemaAuthorizationPolicy (version 1) ---")
	revokeMsg := &cstypes.MsgRevokeSchemaAuthorizationPolicy{
		Corporation: policyAddr,
		Operator:    operatorAddr,
		SchemaId:    schemaID,
		Role:        role,
		Version:     1,
	}
	resp, err = client.BroadcastTx(ctx, operatorAccount, revokeMsg)
	if err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}
	if resp.TxResponse.Code != 0 {
		return fmt.Errorf("step 4 failed code %d: %s", resp.TxResponse.Code, resp.TxResponse.RawLog)
	}
	fmt.Println("OK Step 4: schema authorization policy version 1 revoked")
	waitForTx("revoke cs policy")

	fmt.Println("\n========================================")
	fmt.Println("Journey 311 completed successfully!")
	fmt.Println("CS schema authorization policy: create -> increase version -> revoke")
	fmt.Println("========================================")
	return nil
}
