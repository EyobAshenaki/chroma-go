package db

import (
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

type DataStore struct {
	db *bolt.DB
}

func GetDataStore(dbName string) *DataStore {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	path := fmt.Sprintf("%s.db", dbName)
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	return &DataStore{db: db}
}

func (store *DataStore) Close() {
	store.db.Close()
}

func (store *DataStore) Put(bucketName string, key string, value []byte) error {
	return store.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))

		if err != nil {
			return err
		}

		return bucket.Put([]byte(key), value)
	})
}

func (store *DataStore) Get(bucketName string, key string) ([]byte, error) {
	var value []byte

	err := store.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))

		if bucket == nil {
			return fmt.Errorf("bucket %q not found", bucketName)
		}

		value = bucket.Get([]byte(key))

		if value == nil {
			return fmt.Errorf("key %q not found", key)
		}

		return nil
	})

	return value, err
}
