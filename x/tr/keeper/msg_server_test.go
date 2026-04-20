package keeper_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana/testutil/keeper"
	"github.com/verana-labs/verana/x/tr/keeper"
	"github.com/verana-labs/verana/x/tr/types"
)

// testDigestSRI is a valid sha384 subresource-integrity string reused by
// MsgCreateTrustRegistry / MsgAddGovernanceFrameworkDocument tests to satisfy
// the doc_digest_sri ValidateBasic check introduced in spec draft 13.
const testDigestSRI = "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26"

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context) {
	k, ctx := keepertest.TrustregistryKeeper(t)
	return k, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

func TestMsgServerCreateTrustRegistry(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	// Authority is the group account that will become the controller
	authority := sdk.AccAddress([]byte("test_authority")).String()
	// Operator is the signer authorized to execute on behalf of authority
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"

	testCases := []struct {
		name    string
		msg     *types.MsgCreateTrustRegistry
		isValid bool
	}{
		{
			name: "Valid Create Trust Registry",
			msg: &types.MsgCreateTrustRegistry{
				Corporation:  authority,
				Operator:     operator,
				Did:          validDid,
				Aka:          "http://example.com",
				Language:     "en",
				DocUrl:       "http://example.com/doc-v1",
				DocDigestSri: testDigestSRI,
			},
			isValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.CreateTrustRegistry(ctx, tc.msg)
			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Get ID from DID index
				id, err := k.TrustRegistryDIDIndex.Get(ctx, tc.msg.Did)
				require.NoError(t, err)

				// Get trust registry using ID
				tr, err := k.TrustRegistry.Get(ctx, id)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Did, tr.Did)
				// Verify corporation becomes the controller
				require.Equal(t, tc.msg.Corporation, tr.Corporation)
				require.Equal(t, int32(1), tr.ActiveVersion)
				require.Equal(t, tc.msg.Language, tr.Language)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestMsgServerAddGovernanceFrameworkDocument(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"

	// First, create a trust registry
	createMsg := &types.MsgCreateTrustRegistry{
		Corporation:  authority,
		Operator:     operator,
		Did:          validDid,
		Language:     "en",
		DocUrl:       "http://example.com/doc-v1",
		DocDigestSri: testDigestSRI,
	}
	_, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)

	// Get trust registry ID
	trID, err := k.TrustRegistryDIDIndex.Get(ctx, validDid)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		setupFunc func() // Additional setup for test case
		msg       *types.MsgAddGovernanceFrameworkDocument
		isValid   bool
	}{
		{
			name: "Valid Add Document with Next Version",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc2",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2, // Exactly maxVersion + 1
			},
			isValid: true,
		},
		{
			name: "Valid Add Document to Same Version with Different Language",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "fr",
				Url:         "http://example.com/doc2-fr",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2, // Same version, different language
			},
			isValid: true,
		},
		{
			name: "Valid Add Next Version",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc3",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     3, // Exactly maxVersion + 1
			},
			isValid: true,
		},
		{
			// Spec draft 13 [MOD-TR-MSG-2-2-1]: version MUST be strictly greater
			// than tr.active_version. The active version is immutable — adding
			// documents to v1 (the active version) must be rejected.
			name: "Invalid Add Document to Active Version 1",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc-v1",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     1,
			},
			isValid: false,
		},
		{
			name: "Invalid Trust Registry ID",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        99999,
				Language:    "en",
				Url:         "http://example.com/doc2",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2,
			},
			isValid: false,
		},
		{
			name: "Invalid Language Format",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "invalid-language",
				Url:         "http://example.com/doc2",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2,
			},
			isValid: false,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: "wrong-controller",
				Operator:    "wrong-controller",
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc2",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     2,
			},
			isValid: false,
		},
		{
			name: "Invalid Version (Skipping Version)",
			setupFunc: func() {
				// Add version 3 document first
				msg := &types.MsgAddGovernanceFrameworkDocument{
					Corporation: authority,
					Operator:    operator,
					TrId:        trID,
					Language:    "en",
					Url:         "http://example.com/doc3",
					DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
					Version:     3,
				}
				_, err := ms.AddGovernanceFrameworkDocument(ctx, msg)
				require.NoError(t, err)
			},
			msg: &types.MsgAddGovernanceFrameworkDocument{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Language:    "en",
				Url:         "http://example.com/doc5",
				DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
				Version:     5, // Invalid: should be 4 (maxVersion + 1)
			},
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			resp, err := ms.AddGovernanceFrameworkDocument(ctx, tc.msg)
			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify document was added
				var found bool
				err = k.GFDocument.Walk(ctx, nil, func(id uint64, gfd types.GovernanceFrameworkDocument) (bool, error) {
					if gfd.Language == tc.msg.Language && gfd.Url == tc.msg.Url {
						found = true
						return true, nil
					}
					return false, nil
				})
				require.NoError(t, err)
				require.True(t, found)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestMsgServerIncreaseActiveGovernanceFrameworkVersion(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"

	// Create initial trust registry
	createMsg := &types.MsgCreateTrustRegistry{
		Corporation:  authority,
		Operator:     operator,
		Did:          validDid,
		Language:     "en",
		DocUrl:       "http://example.com/doc-v1",
		DocDigestSri: testDigestSRI,
	}
	_, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)

	// Get trust registry ID
	trID, err := k.TrustRegistryDIDIndex.Get(ctx, validDid)
	require.NoError(t, err)

	// Add version 2 documents
	addGFDocMsg := &types.MsgAddGovernanceFrameworkDocument{
		Corporation: authority,
		Operator:    operator,
		TrId:        trID,
		Language:    "es", // First add Spanish version
		Url:         "http://example.com/doc2-es",
		DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		Version:     2,
	}
	_, err = ms.AddGovernanceFrameworkDocument(ctx, addGFDocMsg)
	require.NoError(t, err)

	// Test cases for version increase
	testCases := []struct {
		name      string
		setupFunc func() // Additional setup for test case
		msg       *types.MsgIncreaseActiveGovernanceFrameworkVersion
		isValid   bool
	}{
		{
			name: "Cannot Increase Version - Missing Default Language Document",
			msg: &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
			},
			isValid: false,
		},
		{
			name: "Valid Version Increase",
			setupFunc: func() {
				// Add English (default language) document for version 2
				msg := &types.MsgAddGovernanceFrameworkDocument{
					Corporation: authority,
					Operator:    operator,
					TrId:        trID,
					Language:    "en",
					Url:         "http://example.com/doc2-en",
					DigestSri:   "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
					Version:     2,
				}
				_, err := ms.AddGovernanceFrameworkDocument(ctx, msg)
				require.NoError(t, err)
			},
			msg: &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
			},
			isValid: true,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: "wrong-controller",
				Operator:    operator,
				TrId:        trID,
			},
			isValid: false,
		},
		{
			name: "Non-existent Trust Registry",
			msg: &types.MsgIncreaseActiveGovernanceFrameworkVersion{
				Corporation: authority,
				Operator:    operator,
				TrId:        99999,
			},
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			resp, err := ms.IncreaseActiveGovernanceFrameworkVersion(ctx, tc.msg)
			if tc.isValid {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify version increase
				tr, err := k.TrustRegistry.Get(ctx, tc.msg.TrId)
				require.NoError(t, err)
				require.Equal(t, int32(2), tr.ActiveVersion)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestMsgServerUpdateTrustRegistry(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	// Create initial trust registry
	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"
	newDid := "did:example:updated987654321"
	createMsg := &types.MsgCreateTrustRegistry{
		Corporation:  authority,
		Operator:     operator,
		Did:          validDid,
		Language:     "en",
		DocUrl:       "http://example.com/doc-v1",
		DocDigestSri: testDigestSRI,
	}
	resp, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Get trust registry ID
	trID, err := k.TrustRegistryDIDIndex.Get(ctx, validDid)
	require.NoError(t, err)

	// Advance block time
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))

	testCases := []struct {
		name      string
		msg       *types.MsgUpdateTrustRegistry
		expectErr bool
	}{
		{
			name: "Valid Update",
			msg: &types.MsgUpdateTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Did:         newDid,
				Aka:         "http://new.example.com",
			},
			expectErr: false,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgUpdateTrustRegistry{
				Corporation: "wrong-controller",
				Operator:    "wrong-controller",
				TrId:        trID,
				Did:         newDid,
				Aka:         "http://example.com",
			},
			expectErr: true,
		},
		{
			name: "Non-existent Trust Registry",
			msg: &types.MsgUpdateTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        99999,
				Did:         newDid,
				Aka:         "http://example.com",
			},
			expectErr: true,
		},
		{
			name: "Clear AKA",
			msg: &types.MsgUpdateTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Did:         newDid,
				Aka:         "", // Empty string to clear AKA
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Advance block time for each test
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			testCtx := sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))

			resp, err := ms.UpdateTrustRegistry(testCtx, tc.msg)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify changes
				tr, err := k.TrustRegistry.Get(testCtx, tc.msg.TrId)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Aka, tr.Aka)
				require.NotEqual(t, tr.Created, tr.Modified)
			}
		})
	}
}

func TestMsgServerArchiveTrustRegistry(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)

	// Create initial trust registry
	authority := sdk.AccAddress([]byte("test_authority")).String()
	operator := sdk.AccAddress([]byte("test_operator")).String()
	validDid := "did:example:123456789abcdefghi"
	createMsg := &types.MsgCreateTrustRegistry{
		Corporation:  authority,
		Operator:     operator,
		Did:          validDid,
		Language:     "en",
		DocUrl:       "http://example.com/doc-v1",
		DocDigestSri: testDigestSRI,
	}
	resp, err := ms.CreateTrustRegistry(ctx, createMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Get trust registry ID
	trID, err := k.TrustRegistryDIDIndex.Get(ctx, validDid)
	require.NoError(t, err)

	// Advance block time
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))

	testCases := []struct {
		name      string
		msg       *types.MsgArchiveTrustRegistry
		expectErr bool
	}{
		{
			name: "Valid Archive",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Archive:     true,
			},
			expectErr: false,
		},
		{
			name: "Already Archived",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Archive:     true,
			},
			expectErr: true,
		},
		{
			// [MOD-TR-MSG-5-3] spec v4 draft 13: archive=false unarchives a currently archived TR.
			// Preceding "Already Archived" run left the TR archived, so this unarchive succeeds.
			name: "Unarchive Succeeds",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Archive:     false,
			},
			expectErr: false,
		},
		{
			// [MOD-TR-MSG-5-2-1] unarchive on a non-archived TR must abort.
			name: "Unarchive Not Archived (abort)",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        trID,
				Archive:     false,
			},
			expectErr: true,
		},
		{
			name: "Wrong Controller",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: "wrong-controller",
				Operator:    operator,
				TrId:        trID,
				Archive:     true,
			},
			expectErr: true,
		},
		{
			name: "Non-existent Trust Registry",
			msg: &types.MsgArchiveTrustRegistry{
				Corporation: authority,
				Operator:    operator,
				TrId:        99999,
				Archive:     true,
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Advance block time for each test
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			testCtx := sdk.WrapSDKContext(sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(time.Hour)))

			resp, err := ms.ArchiveTrustRegistry(testCtx, tc.msg)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify changes
				tr, err := k.TrustRegistry.Get(testCtx, tc.msg.TrId)
				require.NoError(t, err)
				if tc.msg.Archive {
					require.NotNil(t, tr.Archived)
				} else {
					require.Nil(t, tr.Archived)
				}
				require.NotEqual(t, tr.Created, tr.Modified)
			}
		})
	}
}
