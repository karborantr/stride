package simulation

import (
	"math/rand"

	"github.com/Stride-Labs/stride/x/stakeibc/keeper"
	"github.com/Stride-Labs/stride/x/stakeibc/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgSetSafetyGasFeeFlag(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgSetSafetyGasFeeFlag{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the SetSafetyGasFeeFlag simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "SetSafetyGasFeeFlag simulation not implemented"), nil, nil
	}
}