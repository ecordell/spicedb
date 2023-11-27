package memdb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/authzed/spicedb/pkg/datastore"
	"github.com/authzed/spicedb/pkg/datastore/options"
	"github.com/authzed/spicedb/pkg/datastore/test"
	ns "github.com/authzed/spicedb/pkg/namespace"
	nsutil "github.com/authzed/spicedb/pkg/namespace"
	corev1 "github.com/authzed/spicedb/pkg/proto/core/v1"
	"github.com/authzed/spicedb/pkg/tuple"
)

type memDBTest struct{}

func (mdbt memDBTest) New(revisionQuantization, _, gcWindow time.Duration, watchBufferLength uint16) (datastore.Datastore, error) {
	return NewMemdbDatastore(watchBufferLength, revisionQuantization, gcWindow)
}

func TestMemdbDatastore(t *testing.T) {
	test.All(t, memDBTest{})
}

func TestConcurrentWritePanic(t *testing.T) {
	require := require.New(t)

	ds, err := NewMemdbDatastore(0, 1*time.Hour, 1*time.Hour)
	require.NoError(err)

	ctx := context.Background()
	recoverErr := errors.New("panic")

	// Make the namespace very large to increase the likelihood of overlapping
	relationList := make([]*corev1.Relation, 0, 1000)
	for i := 0; i < 1000; i++ {
		relationList = append(relationList, ns.MustRelation(fmt.Sprintf("reader%d", i), nil))
	}

	numPanics := uint64(0)
	require.Eventually(func() bool {
		_, err = ds.ReadWriteTx(ctx, func(ctx context.Context, rwt datastore.ReadWriteTransaction) error {
			g := errgroup.Group{}
			g.Go(func() (err error) {
				defer func() {
					if rec := recover(); rec != nil {
						atomic.AddUint64(&numPanics, 1)
						err = recoverErr
					}
				}()

				return rwt.WriteNamespaces(ctx, ns.Namespace(
					"resource",
					relationList...,
				))
			})

			g.Go(func() (err error) {
				defer func() {
					if rec := recover(); rec != nil {
						atomic.AddUint64(&numPanics, 1)
						err = recoverErr
					}
				}()

				return rwt.WriteNamespaces(ctx, ns.Namespace("user", relationList...))
			})

			return g.Wait()
		})
		return numPanics > 0
	}, 3*time.Second, 10*time.Millisecond)
	require.ErrorIs(err, recoverErr)
}

func TestConcurrentWriteRelsError(t *testing.T) {
	require := require.New(t)

	ds, err := NewMemdbDatastore(0, 1*time.Hour, 1*time.Hour)
	require.NoError(err)

	ctx := context.Background()

	// Kick off a number of writes to ensure at least one hits an error.
	g := errgroup.Group{}

	for i := 0; i < 50; i++ {
		i := i
		g.Go(func() error {
			_, err := ds.ReadWriteTx(ctx, func(ctx context.Context, rwt datastore.ReadWriteTransaction) error {
				updates := []*corev1.RelationTupleUpdate{}
				for j := 0; j < 500; j++ {
					updates = append(updates, &corev1.RelationTupleUpdate{
						Operation: corev1.RelationTupleUpdate_TOUCH,
						Tuple:     tuple.MustParse(fmt.Sprintf("document:doc-%d-%d#viewer@user:tom", i, j)),
					})
				}

				return rwt.WriteRelationships(ctx, updates)
			}, options.WithDisableRetries(true))
			return err
		})
	}

	werr := g.Wait()
	require.Error(werr)
	require.ErrorContains(werr, "serialization max retries exceeded")
}

func TestOptimizedRevisionSnapshotEvaluation(t *testing.T) {
	require := require.New(t)

	ds, err := NewMemdbDatastore(0, 1*time.Second, 1*time.Hour)
	require.NoError(err)

	ctx := context.Background()

	setupDatastore(ds, require)

	head, err := ds.HeadRevision(ctx)
	require.NoError(err)

	// no tuples at head revision, nothing written yet
	tuple0 := readExactTuple(ds.SnapshotReader(head), makeTestTuple(0, 0), require)
	require.Equal("", tuple0)

	// write a tuple
	writeRev, err := ds.ReadWriteTx(ctx, func(ctx context.Context, rwt datastore.ReadWriteTransaction) error {
		return rwt.WriteRelationships(ctx, []*corev1.RelationTupleUpdate{{
			Operation: corev1.RelationTupleUpdate_TOUCH,
			Tuple:     makeTestTuple(0, 0),
		}})
	})
	require.NoError(err)

	// write another tuple
	writeRev2, err := ds.ReadWriteTx(ctx, func(ctx context.Context, rwt datastore.ReadWriteTransaction) error {
		return rwt.WriteRelationships(ctx, []*corev1.RelationTupleUpdate{{
			Operation: corev1.RelationTupleUpdate_TOUCH,
			Tuple:     makeTestTuple(1, 1),
		}})
	})
	require.NoError(err)

	optimizedRev, err := ds.OptimizedRevision(ctx)
	require.NoError(err)
	require.True(optimizedRev.LessThan(writeRev))
	require.True(optimizedRev.LessThan(writeRev2))

	// tuple should be visible at the revision returned by the write
	tuple1AtWriteRev := readExactTuple(ds.SnapshotReader(writeRev2), makeTestTuple(1, 1), require)
	require.Equal(makeTestTuple(1, 1).String(), tuple1AtWriteRev)

	// tuple should not be visible at optimized revision, which is before the write revision
	tuple1AtOptimizedRev := readExactTuple(ds.SnapshotReader(optimizedRev), makeTestTuple(1, 1), require)
	require.Equal("", tuple1AtOptimizedRev)
}

var testResourceNS = nsutil.Namespace(
	testResourceNamespace,
	nsutil.MustRelation(testReaderRelation, nil),
)

var testGroupNS = nsutil.Namespace(
	testGroupNamespace,
	nsutil.MustRelation(testMemberRelation, nil),
)

var testUserNS = nsutil.Namespace(testUserNamespace)

func readExactTuple(reader datastore.Reader, tuple *corev1.RelationTuple, require *require.Assertions) (out string) {
	itr, err := reader.QueryRelationships(context.Background(), tupleToFilter(tuple))
	require.NoError(err)
	defer itr.Close()

	for tpl := itr.Next(); tpl != nil; tpl = itr.Next() {
		if len(out) > 0 {
			require.Fail("found multiple matching exact tuples; test error")
		}
		out = tuple.String()
	}
	return
}

func tupleToFilter(tuple *corev1.RelationTuple) datastore.RelationshipsFilter {
	return datastore.RelationshipsFilter{
		ResourceType:             testResourceNamespace,
		OptionalResourceIds:      []string{tuple.ResourceAndRelation.ObjectId},
		OptionalResourceRelation: tuple.ResourceAndRelation.Relation,
		OptionalSubjectsSelectors: []datastore.SubjectsSelector{{
			OptionalSubjectType: tuple.Subject.Namespace,
			OptionalSubjectIds:  []string{tuple.Subject.ObjectId},
			RelationFilter: datastore.SubjectRelationFilter{
				IncludeEllipsisRelation: true,
			},
		}},
	}
}

func makeTestTuple(resourceID, userID int) *corev1.RelationTuple {
	return &corev1.RelationTuple{
		ResourceAndRelation: &corev1.ObjectAndRelation{
			Namespace: testResourceNamespace,
			ObjectId:  strconv.Itoa(resourceID),
			Relation:  testReaderRelation,
		},
		Subject: &corev1.ObjectAndRelation{
			Namespace: testUserNamespace,
			ObjectId:  strconv.Itoa(userID),
			Relation:  ellipsis,
		},
	}
}

func setupDatastore(ds datastore.Datastore, require *require.Assertions) datastore.Revision {
	ctx := context.Background()

	revision, err := ds.ReadWriteTx(ctx, func(ctx context.Context, rwt datastore.ReadWriteTransaction) error {
		return rwt.WriteNamespaces(ctx, testGroupNS, testResourceNS, testUserNS)
	})
	require.NoError(err)

	return revision
}

const (
	testUserNamespace     = "test/user"
	testResourceNamespace = "test/resource"
	testGroupNamespace    = "test/group"
	testReaderRelation    = "reader"
	testMemberRelation    = "member"
	ellipsis              = "..."
)
