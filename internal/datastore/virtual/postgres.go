package virtual

import (
	"context"
	"fmt"
	"runtime"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"

	v0 "github.com/authzed/authzed-go/proto/authzed/api/v0"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/spicedb/internal/datastore"
)

const (
	errUnableToInstantiate = "unable to instantiate datastore: %w"
	errUnableToQueryTuples = "unable to query tuples: %w"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type ResourceRelation struct {
	resource string
	relation string
}

type TablePk struct {
	table string
	pk string
}

type mapping struct {
	resourceTypeToTable map[string]TablePk
	resourceRelationToJoin map[ResourceRelation]string
	resourceRelationToNamespace map[ResourceRelation]string
}

type virtualPostgres struct {
	conn *pgxpool.Pool
	mapping mapping
}

var _ datastore.Datastore = virtualPostgres{}

func NewVirtualPostgresDatastore(url string) (datastore.Datastore, error) {
	pgxConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	pgxConfig.ConnConfig.Logger = zerologadapter.NewLogger(log.Logger)

	// TODO: options

	dbpool, err := pgxpool.ConnectConfig(context.Background(), pgxConfig)
	if err != nil {
		return nil, fmt.Errorf(errUnableToInstantiate, err)
	}

	return &virtualPostgres{
		conn: dbpool,
		mapping: mapping{
			resourceTypeToTable: map[string]TablePk{
				"customers": {table: "customers", pk: "customer_id"},
				"contacts":  {table: "contacts", pk: "contact_id"},
			},
			resourceRelationToJoin: map[ResourceRelation]string{
				{resource: "contacts", relation: "fk_customer"}: "INNER JOIN customers ON contacts.customer_id=customers.customer_id",
			},
			resourceRelationToNamespace: map[ResourceRelation]string{
				{resource: "contacts", relation: "fk_customer"}: "customers",
			},
		},
	}, nil
}

func (v virtualPostgres) QueryTuples(filter datastore.TupleQueryResourceFilter, revision datastore.Revision) datastore.TupleQuery {
	return virtualPostgresTupleQuery{
		conn:   v.conn,
		filter: filter,
		mapping: v.mapping,
	}
}

func (v virtualPostgres) ReverseQueryTuplesFromSubject(subject *v0.ObjectAndRelation, revision datastore.Revision) datastore.ReverseTupleQuery {
	panic("implement me")
}

func (v virtualPostgres) ReverseQueryTuplesFromSubjectRelation(subjectNamespace, subjectRelation string, revision datastore.Revision) datastore.ReverseTupleQuery {
	panic("implement me")
}

func (v virtualPostgres) ReverseQueryTuplesFromSubjectNamespace(subjectNamespace string, revision datastore.Revision) datastore.ReverseTupleQuery {
	panic("implement me")
}

func (v virtualPostgres) CheckRevision(ctx context.Context, revision datastore.Revision) error {
	panic("implement me")
}

func (v virtualPostgres) WriteTuples(ctx context.Context, preconditions []*v1.Precondition, mutations []*v1.RelationshipUpdate) (datastore.Revision, error) {
	panic("implement me")
}

func (v virtualPostgres) DeleteRelationships(ctx context.Context, preconditions []*v1.Precondition, filter *v1.RelationshipFilter) (datastore.Revision, error) {
	panic("implement me")
}

func (v virtualPostgres) Revision(ctx context.Context) (datastore.Revision, error) {
	panic("implement me")
}

func (v virtualPostgres) SyncRevision(ctx context.Context) (datastore.Revision, error) {
	panic("implement me")
}

func (v virtualPostgres) Watch(ctx context.Context, afterRevision datastore.Revision) (<-chan *datastore.RevisionChanges, <-chan error) {
	panic("implement me")
}

func (v virtualPostgres) WriteNamespace(ctx context.Context, newConfig *v0.NamespaceDefinition) (datastore.Revision, error) {
	panic("implement me")
}

func (v virtualPostgres) ReadNamespace(ctx context.Context, nsName string) (*v0.NamespaceDefinition, datastore.Revision, error) {
	panic("implement me")
}

func (v virtualPostgres) DeleteNamespace(ctx context.Context, nsName string) (datastore.Revision, error) {
	panic("implement me")
}

func (v virtualPostgres) ListNamespaces(ctx context.Context) ([]*v0.NamespaceDefinition, error) {
	panic("implement me")
}

func (v virtualPostgres) IsReady(ctx context.Context) (bool, error) {
	panic("implement me")
}

func (v virtualPostgres) Close() error {
	v.conn.Close()
	return nil
}

type virtualPostgresTupleQuery struct {
	conn   *pgxpool.Pool
	filter datastore.TupleQueryResourceFilter
	mapping mapping
	err    error
}

func (vptq virtualPostgresTupleQuery) Execute(ctx context.Context) (datastore.TupleIterator, error) {
	if vptq.filter.OptionalResourceRelation == "" || vptq.filter.OptionalResourceID == "" {
		panic("implement me")
	}

	// TODO: good
	sql := "SELECT "+vptq.mapping.resourceTypeToTable[vptq.filter.ResourceType].pk +" FROM "+vptq.mapping.resourceTypeToTable[vptq.filter.ResourceType].pk + " " +
			vptq.mapping.resourceRelationToJoin[ResourceRelation{resource: vptq.filter.ResourceType, relation: vptq.filter.OptionalResourceRelation}] + " " +
			"WHERE " +vptq.mapping.resourceTypeToTable[vptq.filter.ResourceType].pk+"=" + vptq.filter.OptionalResourceID
	rows, err := vptq.conn.Query(ctx, sql)
	if err != nil {
		return nil, err
	}

	var tuples []*v0.RelationTuple
	for rows.Next() {
		// TODO: limit
		// if limit > 0 && len(tuples) >= int(limit) {
		// 	return tuples, nil
		// }

		nextTuple := &v0.RelationTuple{
			ObjectAndRelation: &v0.ObjectAndRelation{
				Namespace: vptq.filter.ResourceType,
				ObjectId: vptq.filter.OptionalResourceID,
				Relation: vptq.filter.OptionalResourceRelation,
			},
			User: &v0.User{
				UserOneof: &v0.User_Userset{
					Userset: &v0.ObjectAndRelation{
						Namespace: vptq.mapping.resourceRelationToNamespace[
							ResourceRelation{
								resource: vptq.filter.ResourceType,
								relation: vptq.filter.OptionalResourceRelation,
							}],
						Relation: "...",
					},
				},
			},
		}
		userset := nextTuple.User.GetUserset()
		err := rows.Scan(
			&userset.ObjectId,
		)
		if err != nil {
			return nil, fmt.Errorf(errUnableToQueryTuples, err)
		}

		tuples = append(tuples, nextTuple)
	}
	iter := datastore.NewSliceTupleIterator(tuples)
	runtime.SetFinalizer(iter, datastore.BuildFinalizerFunction())
	return iter, nil
}

func (vptq virtualPostgresTupleQuery) Limit(limit uint64) datastore.CommonTupleQuery {
	panic("implement me")
}

func (vptq virtualPostgresTupleQuery) WithSubjectFilter(filter *v1.SubjectFilter) datastore.TupleQuery {
	panic("implement me")
}

func (vptq virtualPostgresTupleQuery) WithUsersets(usersets []*v0.ObjectAndRelation) datastore.TupleQuery {
	panic("implement me")
}
