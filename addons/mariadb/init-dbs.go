package mariadb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"fastcat.org/go/gdev/resource"
)

func (c svcConfig) initDBs() resource.Resource {
	creds := c.Credentials()
	return &initDBsResource{c, creds}
}

type initDBsResource struct {
	cfg   svcConfig
	creds map[string]string
}

// ID implements resource.Resource.
func (r *initDBsResource) ID() string {
	// TODO: this name stutters
	return fmt.Sprintf("mariadb/%s/init-dbs", r.cfg.name)
}

// Ready implements resource.Resource.
func (r *initDBsResource) Ready(context.Context) (bool, error) {
	// start fails if it can't get ready
	return true, nil
}

// Start implements resource.Resource.
func (r *initDBsResource) Start(ctx context.Context) error {
	if r.cfg.nodePort <= 0 {
		return fmt.Errorf("initializing MariaDB DBs requires enabling mariadb.WithNodePort")
	}

	var user, password string
	for k, v := range r.creds {
		switch k {
		case "MYSQL_USER":
			user = v
		case "MYSQL_PWD":
			password = v
		default:
			return fmt.Errorf("unexpected credential %q", k)
		}
	}
	var dsn string
	if user != "" {
		dsn = user
		if password != "" {
			dsn += ":" + password
		}
		dsn += "@"
	}
	dsn += fmt.Sprintf("tcp(localhost:%d)/", r.cfg.nodePort)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer conn.Close() // nolint:errcheck

	for _, dbName := range r.cfg.initDBNames {
		// create the database if it doesn't exist
		// FIXME: quote this shit better
		_, err := conn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", dbName))
		if err != nil {
			var me *mysql.MySQLError
			if errors.As(err, &me) && string(me.SQLState[:]) == "HY000" {
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
