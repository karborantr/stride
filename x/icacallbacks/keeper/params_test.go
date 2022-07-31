package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/Stride-Labs/stride/testutil/keeper"
	"github.com/Stride-Labs/stride/x/icacallbacks/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.IcacallbacksKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
