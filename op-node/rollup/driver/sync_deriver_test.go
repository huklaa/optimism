package driver

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/node/safedb"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestEngineResetPreservesLocalSafeHistory(t *testing.T) {
	ctx := context.Background()
	logger := testlog.Logger(t, log.LevelError)
	db, err := safedb.NewSafeDB(logger, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	l1At80 := eth.BlockID{Hash: common.Hash{0x40}, Number: 40}
	l2At80 := eth.L2BlockRef{
		Hash:     common.Hash{0x80},
		Number:   80,
		L1Origin: l1At80,
	}
	localSafe := eth.L2BlockRef{
		Hash:     common.Hash{0x64},
		Number:   100,
		L1Origin: eth.BlockID{Hash: common.Hash{0x50}, Number: 50},
	}
	crossSafe := eth.L2BlockRef{
		Hash:     common.Hash{0x28},
		Number:   40,
		L1Origin: eth.BlockID{Hash: common.Hash{0x20}, Number: 20},
	}
	require.NoError(t, db.SafeHeadUpdated(l2At80, l1At80))
	require.NoError(t, db.SafeHeadUpdated(localSafe, localSafe.L1Origin))

	var emitted []event.Event
	d := &SyncDeriver{
		SafeHeadNotifs: db,
		Config:         &rollup.Config{},
		Emitter: event.EmitterFunc(func(_ context.Context, ev event.Event) {
			emitted = append(emitted, ev)
		}),
		Log: logger,
	}
	require.True(t, d.OnEvent(ctx, engine.EngineResetConfirmedEvent{
		LocalSafe: localSafe,
		CrossSafe: crossSafe,
	}))

	actualL1, actualL2, err := db.L1AtSafeHead(ctx, l2At80.Number)
	require.NoError(t, err)
	require.Equal(t, l1At80, actualL1)
	require.Equal(t, l2At80.ID(), actualL2)
	require.Equal(t, []event.Event{derive.ConfirmPipelineResetEvent{}}, emitted)
}
