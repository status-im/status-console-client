package sqlite

import (
	"database/sql"
	"fmt"

	// We require go sqlcipher that overrides default implementation
	"github.com/status-im/migrate/v4"
	"github.com/status-im/migrate/v4/database/sqlcipher"
	bindata "github.com/status-im/migrate/v4/source/go_bindata"
)

// The default number of kdf iterations in sqlcipher (from version 3.0.0)
// https://github.com/sqlcipher/sqlcipher/blob/fda4c68bb474da7e955be07a2b807bda1bb19bd2/CHANGELOG.md#300---2013-11-05
// https://www.zetetic.net/sqlcipher/sqlcipher-api/#kdf_iter
const DefaultKdfIterationsNumber = 64000

// The reduced number of kdf iterations (for performance reasons) which is
// currently used for derivation of the database key
// https://github.com/status-im/status-go/pull/1343
// https://notes.status.im/i8Y_l7ccTiOYq09HVgoFwA
const ReducedKdfIterationsNumber = 3200

func Open(path, key string) (*sql.DB, error) {
	return open(path, key, ReducedKdfIterationsNumber)
}

func OpenWithIter(path, key string, kdfIter int) (*sql.DB, error) {
	return open(path, key, kdfIter)
}

func open(path string, key string, kdfIter int) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	keyString := fmt.Sprintf("PRAGMA key = '%s'", key)

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}

	if _, err = db.Exec(keyString); err != nil {
		return nil, err
	}

	kdfString := fmt.Sprintf("PRAGMA kdf_iter = '%d'", kdfIter)

	if _, err = db.Exec(kdfString); err != nil {
		return nil, err
	}

	// Migrate db
	if err = applyMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}

func applyMigrations(db *sql.DB) error {
	resources := bindata.Resource(
		AssetNames(),
		func(name string) ([]byte, error) {
			return Asset(name)
		},
	)

	source, err := bindata.WithInstance(resources)
	if err != nil {
		return err
	}

	driver, err := sqlcipher.WithInstance(db, &sqlcipher.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"go-bindata",
		source,
		"sqlcipher",
		driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != migrate.ErrNoChange {
		return err
	}

	return nil
}
