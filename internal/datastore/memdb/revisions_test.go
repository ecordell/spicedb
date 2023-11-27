package memdb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHeadRevision(t *testing.T) {
	ds, err := NewMemdbDatastore(0, 0, 500*time.Millisecond)
	require.NoError(t, err)

	older, err := ds.HeadRevision(context.Background())
	require.NoError(t, err)
	err = ds.CheckRevision(context.Background(), older)
	require.NoError(t, err)

	time.Sleep(550 * time.Millisecond)

	// GC window elapsed, last revision is returned even if outside GC window
	newer, err := ds.HeadRevision(context.Background())
	require.NoError(t, err)
	err = ds.CheckRevision(context.Background(), newer)
	require.NoError(t, err)

	require.Equal(t, older, newer)
}

func TestOptimizedRevision(t *testing.T) {
	quantPeriod := 500 * time.Millisecond

	ds, err := NewMemdbDatastore(0, quantPeriod, 1*time.Hour)
	require.NoError(t, err)

	head, err := ds.HeadRevision(context.Background())
	require.NoError(t, err)

	rev1, err := ds.OptimizedRevision(context.Background())
	require.NoError(t, err)

	require.True(t, rev1.LessThan(head))

	time.Sleep(quantPeriod)
	rev2, err := ds.OptimizedRevision(context.Background())
	require.NoError(t, err)

	require.True(t, rev1.LessThan(rev2))
	require.True(t, head.LessThan(rev2))
}

func (mdb *memdbDatastore) ExampleRetryableError() error {
	return errSerialization
}
