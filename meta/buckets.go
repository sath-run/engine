package meta

import bolt "go.etcd.io/bbolt"

var (
	bucketKeyVersion    = []byte(schemaVersion)
	bucketKeyDBVersion  = []byte("version") // stores the version of the schema
	bucketKeyCredential = []byte("credential")

	bucketKeyUserToken   = []byte("usertoken")
	bucketKeyDeviceToken = []byte("devicetoken")
)

func getBucket(tx *bolt.Tx, keys ...[]byte) *bolt.Bucket {
	bkt := tx.Bucket(keys[0])

	for _, key := range keys[1:] {
		if bkt == nil {
			break
		}
		bkt = bkt.Bucket(key)
	}

	return bkt
}

func createBucketIfNotExists(tx *bolt.Tx, keys ...[]byte) (*bolt.Bucket, error) {
	bkt, err := tx.CreateBucketIfNotExists(keys[0])
	if err != nil {
		return nil, err
	}

	for _, key := range keys[1:] {
		bkt, err = bkt.CreateBucketIfNotExists(key)
		if err != nil {
			return nil, err
		}
	}

	return bkt, nil
}

func credentialBucketPath() [][]byte {
	return [][]byte{bucketKeyVersion, bucketKeyCredential}
}

func getCredentialBucket(tx *bolt.Tx) *bolt.Bucket {
	return getBucket(tx, credentialBucketPath()...)
}
