package simulation

import (
	"math/rand"

	"github.com/Stride-Labs/stride/x/interchainquery/keeper"
	"github.com/Stride-Labs/stride/x/interchainquery/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgQueryDelegatedbalance(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgQueryDelegatedbalance{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the QueryDelegatedbalance simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "QueryDelegatedbalance simulation not implemented"), nil, nil
	}
}