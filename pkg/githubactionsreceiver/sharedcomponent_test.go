package githubactionsreceiver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

type mockComponent struct {
	component.StartFunc
	component.ShutdownFunc
}

func TestNewSharedComponents(t *testing.T) {
	sharedComps := NewSharedComponents[int, mockComponent]()
	assert.NotNil(t, sharedComps)
	assert.Empty(t, sharedComps.comps)
}

func TestSharedComponents_GetOrAdd(t *testing.T) {
	nop := &mockComponent{}
	createNop := func() (mockComponent, error) { return *nop, nil }

	key := component.NewID("test")
	comps := NewSharedComponents[component.ID, mockComponent]()
	got, err := comps.GetOrAdd(key, createNop)
	assert.NoError(t, err)
	assert.Len(t, comps.comps, 1)
	assert.Equal(t, nop, &got.component)

	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Len(t, comps.comps, 0)
	newGot, err := comps.GetOrAdd(key, createNop)
	assert.NotSame(t, got, newGot)
}

func TestSharedComponent(t *testing.T) {
	wantErr := errors.New("err")
	startCount := 0
	stopCount := 0
	mockComp := &mockComponent{
		StartFunc: func(ctx context.Context, host component.Host) error {
			startCount++
			return wantErr
		},
		ShutdownFunc: func(ctx context.Context) error {
			stopCount++
			return wantErr
		},
	}
	createComp := func() (mockComponent, error) { return *mockComp, nil }

	comps := NewSharedComponents[component.ID, mockComponent]()
	key := component.NewID("test")
	got, err := comps.GetOrAdd(key, createComp)
	assert.NoError(t, err)
	assert.Equal(t, wantErr, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, startCount)

	assert.NoError(t, got.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, 1, startCount)
	assert.Equal(t, wantErr, got.Shutdown(context.Background()))
	assert.Equal(t, 1, stopCount)

	assert.NoError(t, got.Shutdown(context.Background()))
	assert.Equal(t, 1, stopCount)
}
