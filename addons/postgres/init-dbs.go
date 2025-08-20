package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"fastcat.org/go/gdev/resource"
)

func (c pgSvcConfig) initDBs() resource.Resource {
	creds := c.Credentials()
	return &initDBsResource{c, creds}
}

type initDBsResource struct {
	cfg   pgSvcConfig
	creds map[string]string
}

// ID implements resource.Resource.
func (r *initDBsResource) ID() string {
	// TODO: this name stutters
	return fmt.Sprintf("postgres/%s/init-dbs", r.cfg.name)
}

// Ready implements resource.Resource.
func (r *initDBsResource) Ready(context.Context) (bool, error) {
	// start fails if it can't get ready
	return true, nil
}

// Start implements resource.Resource.
func (r *initDBsResource) Start(ctx context.Context) error {
	if r.cfg.nodePort <= 0 {
		return fmt.Errorf("initializing PG DBs requires enabling postgres.WithNodePort")
	}

	cc, err := pgx.ParseConfig(fmt.Sprintf("postgres://localhost:%d/postgres", r.cfg.nodePort))
	if err != nil {
		return err
	}
	// would be nice if pgx provided an override for os.Getenv to help with this
	for k, v := range r.creds {
		switch k {
		case "PGUSER":
			cc.User = v
		case "PGPASSWORD":
			cc.Password = v
		default:
			return fmt.Errorf("unexpected credential %q", k)
		}
	}
	conn, err := pgx.ConnectConfig(ctx, cc)
	if err != nil {
		return err
	}
	defer conn.Close(ctx) // nolint:errcheck

	for _, dbName := range r.cfg.initDBNames {
		safeName := pgx.Identifier{dbName}.Sanitize()
		// create the database if it doesn't exist
		_, err := conn.Exec(ctx, "CREATE DATABASE "+safeName)
		if err != nil {
			var pge *pgconn.PgError
			if errors.As(err, &pge) && pge.SQLState() == "42P04" {
				// database already exists
				continue
			}
			return fmt.Errorf("error creating database %q: %w", dbName, err)
		}
	}

	return nil
}

// Stop implements resource.Resource.
func (r *initDBsResource) Stop(context.Context) error {
	// no-op
	return nil
}
