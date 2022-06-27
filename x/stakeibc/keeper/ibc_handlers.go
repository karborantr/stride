package keeper

import (
	"encoding/json"
	"fmt"

	epochtypes "github.com/Stride-Labs/stride/x/epochs/types"
	recordstypes "github.com/Stride-Labs/stride/x/records/types"
	"github.com/Stride-Labs/stride/x/stakeibc/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	"github.com/golang/protobuf/proto"

	icatypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
)

// Implements core logic for OnAcknowledgementPacket
// TODO(TEST-28): Add ack handling logic for various ICA calls
// TODO(TEST-33): Scope out what to store on different acks (by function call, success/failure)
func (k Keeper) HandleAcknowledgement(ctx sdk.Context, modulePacket channeltypes.Packet, acknowledgement []byte) error {
	ack := channeltypes.Acknowledgement_Result{}
	var eventType string
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(types.AttributeKeyAck, fmt.Sprintf("%v", ack)),
		),
	)
	err := json.Unmarshal(acknowledgement, &ack)
	if err != nil {
		ackErr := channeltypes.Acknowledgement_Error{}
		err := json.Unmarshal(acknowledgement, &ackErr)
		if err != nil {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					eventType,
					sdk.NewAttribute(types.AttributeKeyAckError, ackErr.Error),
				),
			)
			k.Logger(ctx).Error("Unable to unmarshal acknowledgement error", "error", err, "data", acknowledgement)
			return err
		}
		k.Logger(ctx).Error("Unable to unmarshal acknowledgement result", "error", err, "remote_err", ackErr, "data", acknowledgement)
		return err
	}

	

	txMsgData := &sdk.TxMsgData{}
	err = proto.Unmarshal(ack.Result, txMsgData)
	if err != nil {
		k.Logger(ctx).Error("Unable to unmarshal acknowledgement", "error", err, "ack", ack.Result)
		return err
	}

	var packetData icatypes.InterchainAccountPacketData
	err = icatypes.ModuleCdc.UnmarshalJSON(modulePacket.GetData(), &packetData)
	if err != nil {
		k.Logger(ctx).Error("unable to unmarshal acknowledgement packet data", "error", err, "data", packetData)
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal packet data: %s", err.Error())
	}

	msgs, err := icatypes.DeserializeCosmosTx(k.cdc, packetData.Data)
	if err != nil {
		k.Logger(ctx).Error("unable to decode messages", "err", err)
		return err
	}

	for msgIndex, msgData := range txMsgData.Data {
		src := msgs[msgIndex]
		pstr := fmt.Sprintf("\t[DOGE] Message {%s}", msgData.MsgType)
		k.Logger(ctx).Info(pstr)
		switch msgData.MsgType {
		// staking to validators
		case "/cosmos.staking.v1beta1.MsgDelegate":
			response := stakingtypes.MsgDelegateResponse{}
			err := proto.Unmarshal(msgData.Data, &response)
			if err != nil {
				k.Logger(ctx).Error("unable to unmarshal MsgDelegate response", "error", err)
				return err
			}
			k.Logger(ctx).Info("Delegated", "response", response)
			// we should update delegation records here.
			if err := k.HandleDelegate(ctx, src); err != nil {
				return err
			}
			continue
		// unstake
		case "/cosmos.staking.v1beta1.MsgUndelegate":
			response := stakingtypes.MsgUndelegateResponse{}
			err := proto.Unmarshal(msgData.Data, &response)
			if err != nil {
				k.Logger(ctx).Error("Unable to unmarshal MsgUndelegate response", "error", err)
				return err
			}
			k.Logger(ctx).Debug("Delegated", "response", response)
			// we should update delegation records here.
			if err := k.HandleUndelegate(ctx, src); err != nil {
				return err
			}
			continue
		// withdrawing rewards ()
		case "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward":
			// TODO: Implement! (lo pri)
			continue
		case "/cosmos.bank.v1beta1.MsgSend":
			response := banktypes.MsgSendResponse{}
			err := proto.Unmarshal(msgData.Data, &response)
			if err != nil {
				k.Logger(ctx).Error("unable to unmarshal MsgSend response", "error", err)
				return err
			}
			k.Logger(ctx).Info("Sent", "response", response)

			// we should update delegation records here.
			if err := k.HandleSend(ctx, src); err != nil {
				return err
			}
			continue
		default:
			k.Logger(ctx).Error("Unhandled acknowledgement packet", "type", msgData.MsgType)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute(types.AttributeKeyAckSuccess, string(ack.Result)),
		),
	)
	return nil
}


func (k *Keeper) HandleSend(ctx sdk.Context, msg sdk.Msg) error {
	k.Logger(ctx).Info("Received MsgSend acknowledgement")
	// first, type assertion. we should have banktypes.MsgSend
	sendMsg, ok := msg.(*banktypes.MsgSend)
	if !ok {
		k.Logger(ctx).Error("unable to cast source message to MsgSend")
		return fmt.Errorf("unable to cast source message to MsgSend")
	}
	coin := sendMsg.Amount[0]
	// Pull host zone
	hostZoneDenom := coin.Denom
	amount := coin.Amount.Int64()
	zone, err := k.GetHostZoneFromHostDenom(ctx, hostZoneDenom)
	if err != nil {
		return err
	}

	// Check to and from addresses
	withdrawalAddress := zone.GetWithdrawalAccount().GetAddress()
	delegationAddress := zone.GetDelegationAccount().GetAddress()

	// Only process bank sends that reinvest user rewards
	if sendMsg.FromAddress == withdrawalAddress && sendMsg.ToAddress == delegationAddress {
		// fetch epoch
		strideEpochTracker, found := k.GetEpochTracker(ctx, epochtypes.STRIDE_EPOCH)
		if !found {
			k.Logger(ctx).Info("failed to find epoch")
			return sdkerrors.Wrapf(types.ErrInvalidLengthEpochTracker, "no number for epoch (%s)", epochtypes.STRIDE_EPOCH)
		}
		epochNumber := strideEpochTracker.EpochNumber
		// create a new record so that rewards are reinvested
		record := recordstypes.DepositRecord{
			Id:         0,
			Amount:     amount,
			Denom:      hostZoneDenom,
			HostZoneId: zone.ChainId,
			Status:    recordstypes.DepositRecord_STAKE,
			Source: recordstypes.DepositRecord_WITHDRAWAL_ICA,
			EpochNumber: uint64(epochNumber),
		}
		k.RecordsKeeper.AppendDepositRecord(ctx, record)	
	} else {
		return nil
	}


	return nil
}


func (k *Keeper) HandleDelegate(ctx sdk.Context, msg sdk.Msg) error {
	k.Logger(ctx).Info("Received MsgDelegate acknowledgement")
	// first, type assertion. we should have stakingtypes.MsgDelegate
	delegateMsg, ok := msg.(*stakingtypes.MsgDelegate)
	if !ok {
		k.Logger(ctx).Error("unable to cast source message to MsgDelegate")
		return fmt.Errorf("unable to cast source message to MsgDelegate")
	}
	// CHECK ZONE
	hostZoneDenom := delegateMsg.Amount.Denom
	amount := delegateMsg.Amount.Amount.Int64()
	zone, err := k.GetHostZoneFromHostDenom(ctx, hostZoneDenom)
	if err != nil {
		return err
	}
	record, found := k.RecordsKeeper.GetStakeDepositRecordByAmount(ctx, amount, zone.ChainId)
	if found != true {
		return sdkerrors.Wrapf(sdkerrors.ErrNotFound, "No deposit record found for zone: %s, amount: %s", zone.ChainId, amount)
	}

	k.RecordsKeeper.RemoveDepositRecord(ctx, record.Id)
	return nil
}


// TODO(TEST-28): Burn stAssets if RedeemStake succeeds
func (k Keeper) HandleUndelegate(ctx sdk.Context, msg sdk.Msg) error {
	k.Logger(ctx).Info("Received MsgUndelegate acknowledgement")
	// first, type assertion. we should have stakingtypes.MsgDelegate
	undelegateMsg, ok := msg.(*stakingtypes.MsgUndelegate)
	_ = undelegateMsg
	if !ok {
		k.Logger(ctx).Error("unable to cast source message to MsgUndelegate")
		return fmt.Errorf("unable to cast source message to MsgUndelegate")
	}

	// Implement!
	// burn stAssets if successful
	// return stAssets to user if unsuccessful

	return nil
}
