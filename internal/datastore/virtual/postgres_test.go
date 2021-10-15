package virtual

import (
	"context"
	"fmt"
	v0 "github.com/authzed/authzed-go/proto/authzed/api/v0"
	"log"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/ory/dockertest"
	"github.com/stretchr/testify/require"

	"github.com/authzed/spicedb/internal/datastore"
	"github.com/authzed/spicedb/pkg/secrets"
)

func postgres(creds string, portNum uint16) (*pgxpool.Pool, string, func()) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "11.13",
		Env:        []string{"POSTGRES_PASSWORD=secret", "POSTGRES_DB=defaultdb"},
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	var dbpool *pgxpool.Pool
	port := resource.GetPort(fmt.Sprintf("%d/tcp", portNum))
	if err = pool.Retry(func() error {
		var err error
		dbpool, err = pgxpool.Connect(context.Background(), fmt.Sprintf("postgres://%s@localhost:%s/defaultdb?sslmode=disable", creds, port))
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	cleanup := func() {
		if err = pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}

	return dbpool, port, cleanup
}

func newTestDatastore(require *require.Assertions, pool *pgxpool.Pool, creds string, port string) datastore.Datastore {
	uniquePortion, err := secrets.TokenHex(4)
	require.NoError(err)
	newDBName := "db" + uniquePortion
	_, err = pool.Exec(context.Background(), "CREATE DATABASE "+newDBName)
	require.NoError(err)

	// fill with test data
	schema := `
CREATE TABLE customers(
   customer_id INT GENERATED ALWAYS AS IDENTITY,
   customer_name VARCHAR(255) NOT NULL,
   PRIMARY KEY(customer_id)
);

CREATE TABLE contacts(
   contact_id INT GENERATED ALWAYS AS IDENTITY,
   customer_id INT,
   contact_name VARCHAR(255) NOT NULL,
   phone VARCHAR(15),
   email VARCHAR(100),
   PRIMARY KEY(contact_id),
   CONSTRAINT fk_customer
      FOREIGN KEY(customer_id) 
	  REFERENCES customers(customer_id)
	  ON DELETE CASCADE
);

INSERT INTO customers(customer_name)
VALUES('BigCo'),
      ('SmallFry');	   
	   
INSERT INTO contacts(customer_id, contact_name, phone, email)
VALUES(1,'John Doe','(408)-111-1234','john.doe@bigco.dev'),
      (1,'Jane Doe','(408)-111-1235','jane.doe@bigco.dev'),
      (2,'Jeshk Doe','(408)-222-1234','jeshk.doe@smallfry.dev');
`

	_, err = pool.Exec(context.Background(), schema)
	require.NoError(err)

	connectStr := fmt.Sprintf(
		"postgres://%s@localhost:%s/%s?sslmode=disable",
		creds,
		port,
		newDBName,
	)
	ds, err := NewVirtualPostgresDatastore(connectStr)
	require.NoError(err)

	return ds
}

func TestPostgresDatastore(t *testing.T) {
	pg, port, cleanup := postgres("postgres:secret", 5432)
	defer cleanup()
	require := require.New(t)
	ds := newTestDatastore(require, pg, "postgres:secret", port)
	iter, err := ds.QueryTuples(datastore.TupleQueryResourceFilter{
		ResourceType:             "contacts",
		OptionalResourceID:       "1",
		OptionalResourceRelation: "fk_customer",
	}, datastore.NoRevision).Execute(context.TODO())
	require.NoError(err)

	// TODO: use tuplechecker
	tuples := make([]*v0.RelationTuple, 0)
	for t := iter.Next(); t != nil; t = iter.Next() {
		tuples = append(tuples, t)
	}
	fmt.Println(t)
}
