package meta

import (
	"github.com/sath-run/engine/constants"
	bolt "go.etcd.io/bbolt"
)

func getCredentialValue(key []byte) (string, error) {
	var token string
	err := db.View(func(tx *bolt.Tx) error {
		bkt := getCredentialBucket(tx)
		v := bkt.Get(key)
		token = string(v)
		return nil
	})
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", constants.ErrNil
	}
	return token, nil
}

func setCredentialValue(key []byte, value string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bkt := getCredentialBucket(tx)
		return bkt.Put(key, []byte(value))
	})
	return err
}

func removeCredentialValue(key []byte) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bkt := getCredentialBucket(tx)
		return bkt.Delete(key)
	})
	return err
}

func GetCredentialUserToken() (string, error) {
	return getCredentialValue(bucketKeyUserToken)
}

func SetCredentialUserToken(token string) error {
	return setCredentialValue(bucketKeyUserToken, token)
}

func RemoveCredentialUserToken() error {
	return removeCredentialValue(bucketKeyUserToken)
}

func GetCredentialDeviceToken() (string, error) {
	return getCredentialValue(bucketKeyDeviceToken)
}

func SetCredentialDeviceToken(token string) error {
	return setCredentialValue(bucketKeyDeviceToken, token)
}
