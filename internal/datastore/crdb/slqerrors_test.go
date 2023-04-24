//go:build ci && docker
// +build ci,docker

package crdb

import (
	"github.com/authzed/spicedb/pkg/namespace"
)

const (
	testUserNamespace     = "test/user"
	testResourceNamespace = "test/resource"
	testGroupNamespace    = "test/group"
	testReaderRelation    = "reader"
	testMemberRelation    = "member"
)

var testResourceNS = namespace.Namespace(
	testResourceNamespace,
	namespace.MustRelation(testReaderRelation, nil),
)

var testGroupNS = namespace.Namespace(
	testGroupNamespace,
	namespace.MustRelation(testMemberRelation, nil),
)

var testUserNS = namespace.Namespace(testUserNamespace)

// func executeWithErrors(errors *[]error, maxRetries uint8) executeTxRetryFunc {
// 	return func(ctx context.Context, fn innerFunc) (err error) {
// 		wrappedFn := func(ctx context.Context) error {
// 			err = fn(ctx)
// 			if err != nil {
// 				return err
// 			}
// 			if len(*errors) > 0 {
// 				retErr := (*errors)[0]
// 				(*errors) = (*errors)[1:]
// 				return retErr
// 			}
// 			return nil
// 		}
//
// 		return executeWithResets(ctx, wrappedFn, maxRetries)
// 	}
// }
//
// func TestTxReset(t *testing.T) {
// 	b := testdatastore.RunCRDBForTesting(t, "")
// 	log.SetGlobalLogger(zerolog.New(os.Stdout))
//
// 	cases := []struct {
// 		name        string
// 		maxRetries  uint8
// 		errors      []error
// 		expectError bool
// 	}{
// 		{
// 			name:       "retryable",
// 			maxRetries: 4,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name:       "resettable",
// 			maxRetries: 5,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbAmbiguousErrorCode},
// 				&pgconn.PgError{Code: crdbAmbiguousErrorCode},
// 				&pgconn.PgError{Code: crdbServerNotAcceptingClients},
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name:       "mixed",
// 			maxRetries: 50,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 				&pgconn.PgError{Code: crdbAmbiguousErrorCode},
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name:        "noErrors",
// 			maxRetries:  50,
// 			errors:      []error{},
// 			expectError: false,
// 		},
// 		{
// 			name:       "nonRecoverable",
// 			maxRetries: 1,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 				&pgconn.PgError{Code: crdbAmbiguousErrorCode},
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name:       "stale connections",
// 			maxRetries: 4,
// 			errors: []error{
// 				errors.New("unexpected EOF"),
// 				errors.New("broken pipe"),
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name:       "clockSkew",
// 			maxRetries: 2,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbUnknownSQLState, Message: crdbClockSkewMessage},
// 			},
// 			expectError: false,
// 		},
// 	}
// 	for _, tt := range cases {
// 		t.Run(tt.name, func(t *testing.T) {
// 			require := require.New(t)
//
// 			ds := b.NewDatastore(t, func(engine, uri string) datastore.Datastore {
// 				ds, err := newCRDBDatastore(
// 					uri,
// 					GCWindow(24*time.Hour),
// 					RevisionQuantization(5*time.Second),
// 					WatchBufferLength(128),
// 					WriteConnsMinOpen(3),
// 					ReadConnsMinOpen(3),
// 				)
// 				require.NoError(err)
// 				return ds
// 			})
// 			ctx := context.Background()
// 			r, err := ds.ReadyState(ctx)
// 			require.NoError(err)
// 			require.True(r.IsReady)
//
// 			rev, err := ds.ReadWriteTx(ctx, func(rwt datastore.ReadWriteTransaction) error {
// 				return rwt.WriteNamespaces(ctx, testGroupNS, testResourceNS, testUserNS)
// 			})
// 			require.NoError(err)
// 			ds.(*crdbDatastore).execute = executeWithErrors(&tt.errors, tt.maxRetries)
// 			defer ds.Close()
//
// 			// WriteNamespace utilizes execute so we'll use it
// 			rev, err = ds.ReadWriteTx(ctx, func(rwt datastore.ReadWriteTransaction) error {
// 				update := &core.RelationTupleUpdate{
// 					Operation: core.RelationTupleUpdate_TOUCH,
// 					Tuple: &core.RelationTuple{
// 						ResourceAndRelation: &core.ObjectAndRelation{
// 							Namespace: testResourceNamespace,
// 							ObjectId:  "res",
// 							Relation:  testReaderRelation,
// 						},
// 						Subject: &core.ObjectAndRelation{
// 							Namespace: testUserNamespace,
// 							ObjectId:  "user",
// 							Relation:  "...",
// 						},
// 					},
// 				}
// 				return rwt.WriteRelationships(ctx, []*core.RelationTupleUpdate{
// 					update,
// 				})
// 			})
// 			if tt.expectError {
// 				require.Error(err)
// 				require.Equal(datastore.NoRevision, rev)
// 			} else {
// 				require.NoError(err)
// 				require.True(rev.GreaterThan(revision.NoRevision))
// 			}
// 		})
// 	}
// }
//
// func TestTxResetReads(t *testing.T) {
// 	b := testdatastore.RunCRDBForTesting(t, "")
// 	log.SetGlobalLogger(zerolog.New(os.Stdout))
//
// 	cases := []struct {
// 		name        string
// 		maxRetries  uint8
// 		errors      []error
// 		expectError bool
// 	}{
// 		{
// 			name:       "retryable",
// 			maxRetries: 7,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name:       "resettable",
// 			maxRetries: 7,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbAmbiguousErrorCode},
// 				&pgconn.PgError{Code: crdbAmbiguousErrorCode},
// 				&pgconn.PgError{Code: crdbServerNotAcceptingClients},
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name:       "mixed",
// 			maxRetries: 50,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 				&pgconn.PgError{Code: crdbAmbiguousErrorCode},
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name:        "noErrors",
// 			maxRetries:  50,
// 			errors:      []error{},
// 			expectError: false,
// 		},
// 		{
// 			name:       "nonRecoverable",
// 			maxRetries: 2,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbRetryErrCode},
// 				&pgconn.PgError{Code: crdbAmbiguousErrorCode},
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name:       "stale connections",
// 			maxRetries: 5,
// 			errors: []error{
// 				errors.New("unexpected EOF"),
// 				errors.New("broken pipe"),
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name:       "clockSkew",
// 			maxRetries: 2,
// 			errors: []error{
// 				&pgconn.PgError{Code: crdbUnknownSQLState, Message: crdbClockSkewMessage},
// 			},
// 			expectError: false,
// 		},
// 	}
// 	for _, tt := range cases {
// 		t.Run(tt.name, func(t *testing.T) {
// 			require := require.New(t)
//
// 			ds := b.NewDatastore(t, func(engine, uri string) datastore.Datastore {
// 				ds, err := newCRDBDatastore(
// 					uri,
// 					GCWindow(24*time.Hour),
// 					RevisionQuantization(5*time.Second),
// 					WatchBufferLength(128),
// 					WriteConnsMaxOpen(3),
// 					ReadConnsMinOpen(3),
// 				)
// 				require.NoError(err)
// 				return ds
// 			})
// 			ds.(*crdbDatastore).execute = executeWithErrors(&tt.errors, tt.maxRetries)
// 			defer ds.Close()
//
// 			ctx := context.Background()
// 			r, err := ds.ReadyState(ctx)
// 			require.NoError(err)
// 			require.True(r.IsReady)
//
// 			read := func() (datastore.Revision, error) {
// 				rev, err := ds.HeadRevision(ctx)
// 				if err != nil {
// 					return datastore.NoRevision, err
// 				}
// 				_, err = ds.SnapshotReader(rev).ListAllNamespaces(ctx)
// 				return rev, err
// 			}
//
// 			rev, err := read()
// 			if tt.expectError {
// 				require.Error(err)
// 				require.Equal(datastore.NoRevision, rev)
// 			} else {
// 				require.NoError(err)
// 				require.True(rev.GreaterThan(revision.NoRevision))
// 			}
// 		})
// 	}
// }
