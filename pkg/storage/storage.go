package storage

import (
	"encoding/json"
	"fmt"
	"github.com/xwi88/bbolt"
	"log"
)

func InitialiseDB(path string) (*bbolt.DB, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("peers"))
		if err != nil {
			return fmt.Errorf("DB: Could not create peers bucket: %v", err)
		} else {
			log.Println("DB: Bucket created was created")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Println("DB: Successfully initialised...")

	return db, nil
}

func CachePeerToDB(db *bbolt.DB, peer string) error {
	entry := peer
	entryBytes, err := json.Marshal(entry)

	if err != nil {
		return err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		err := tx.Bucket([]byte("peers")).Put([]byte("peer"), (entryBytes))
		if err != nil {
			return err
		}

		return nil
	})

	//log.Println("Peer added to cache", entry)
	defer db.Close()

	return err
}

func LoadPeersFromDB(db *bbolt.DB) error {
	err := db.View(func(tx *bbolt.Tx) error {
		peers := tx.Bucket([]byte("peers")).Get([]byte("peer"))
		log.Println("Peers:\n", string(peers))
		return nil
	})

	if err != nil {
		return err
	}

	defer db.Close()

	return nil
}