package meta

import (
	"context"
	"encoding/binary"
	"fmt"
	"path/filepath"

	"github.com/sath-run/engine/utils"
	bolt "go.etcd.io/bbolt"
)

const (
	// schemaVersion represents the schema version of
	// the database. This schema version represents the
	// structure of the data in the database. The schema
	// can envolve at any time but any backwards
	// incompatible changes or structural changes require
	// bumping the schema version.
	schemaVersion = "v0"

	// dbVersion represents updates to the schema
	// version which are additions and compatible with
	// prior version of the same schema.
	dbVersion = 1
)

type DB struct {
	db *bolt.DB
}

var db *DB

func Init() error {
	options := *bolt.DefaultOptions

	// Reading bbolt's freelist sometimes fails when the file has a data corruption.
	// Disabling freelist sync reduces the chance of the breakage.
	// https://github.com/etcd-io/bbolt/pull/1
	// https://github.com/etcd-io/bbolt/pull/6
	options.NoFreelistSync = true

	path := filepath.Join(utils.SathHome, "meta.db")
	bdb, err := bolt.Open(path, 0644, &options)
	if err != nil {
		return err
	}
	db = NewDB(bdb)
	if err := db.Init(context.TODO()); err != nil {
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := createBucketIfNotExists(tx, credentialBucketPath()...); err != nil {
			return err
		}
		return nil
	})
	return err
}

func NewDB(db *bolt.DB) *DB {
	return &DB{db: db}
}

func (m *DB) Init(ctx context.Context) error {
	err := m.db.Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists(bucketKeyVersion)
		if err != nil {
			return err
		}

		versionEncoded, err := encodeInt(dbVersion)
		if err != nil {
			return err
		}

		return bkt.Put(bucketKeyDBVersion, versionEncoded)
	})
	return err
}

func encodeInt(i int64) ([]byte, error) {
	var (
		buf      [binary.MaxVarintLen64]byte
		iEncoded = buf[:]
	)
	iEncoded = iEncoded[:binary.PutVarint(iEncoded, i)]

	if len(iEncoded) == 0 {
		return nil, fmt.Errorf("failed encoding integer = %v", i)
	}
	return iEncoded, nil
}

// View runs a readonly transaction on the metadata store.
func (m *DB) View(fn func(*bolt.Tx) error) error {
	return m.db.View(fn)
}

// Update runs a writable transaction on the metadata store.
func (m *DB) Update(fn func(*bolt.Tx) error) error {
	err := m.db.Update(fn)
	return err
}
