package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/di/keeper"
	"github.com/verana-labs/verana/x/di/types"
)

func TestGetDigest_NilRequest(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	resp, err := qs.GetDigest(f.ctx, nil)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "invalid request")
}

func TestGetDigest_EmptyDigest(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	resp, err := qs.GetDigest(f.ctx, &types.QueryGetDigestRequest{
		Digest: "",
	})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "digest must not be empty")
}

func TestGetDigest_NotFound(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	resp, err := qs.GetDigest(f.ctx, &types.QueryGetDigestRequest{
		Digest: "sha256-does-not-exist",
	})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "digest not found")
}

func TestGetDigest_Success(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)

	digestStr := "sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26"
	created := time.Date(2025, 1, 14, 19, 40, 37, 967000000, time.UTC)

	// Store a digest directly in the collection.
	err := f.keeper.Digests.Set(f.ctx, digestStr, types.Digest{
		Digest:  digestStr,
		Created: created,
	})
	require.NoError(t, err)

	// Query the digest.
	resp, err := qs.GetDigest(f.ctx, &types.QueryGetDigestRequest{
		Digest: digestStr,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Digest)
	require.Equal(t, digestStr, resp.Digest.Digest)
	require.Equal(t, created, resp.Digest.Created)
}
