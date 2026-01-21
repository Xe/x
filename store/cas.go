package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

func NewCAS(baseDir string) (Interface, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("%w: base directory cannot be empty", ErrBadConfig)
	}

	objectsDir := filepath.Join(baseDir, "objects")
	if err := os.MkdirAll(objectsDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: can't create objects directory: %w", ErrBadConfig, err)
	}

	dbPath := filepath.Join(baseDir, "index.bolt")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: can't open bolt db: %w", ErrBadConfig, err)
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("keys"))
		return err
	}); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: can't create bucket: %w", ErrBadConfig, err)
	}

	return &CAS{
		base:       baseDir,
		objectsDir: objectsDir,
		db:         db,
	}, nil
}

type CAS struct {
	base       string
	objectsDir string
	db         *bolt.DB
}

func (c *CAS) Close() error {
	return c.db.Close()
}

func (c *CAS) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	err := c.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("keys"))
		if bucket == nil {
			return fmt.Errorf("%w: bucket not found", ErrNotFound)
		}

		if bucket.Get([]byte(key)) == nil {
			return fmt.Errorf("%w: key not found", ErrNotFound)
		}

		return bucket.Delete([]byte(key))
	})

	if err != nil {
		return err
	}

	iopsMetrics.WithLabelValues("cas", "Delete")
	return nil
}

func (c *CAS) Exists(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	err := c.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("keys"))
		if bucket == nil {
			return fmt.Errorf("%w: bucket not found", ErrNotFound)
		}

		val := bucket.Get([]byte(key))
		if val == nil {
			return fmt.Errorf("%w: key not found", ErrNotFound)
		}

		return nil
	})

	if err != nil {
		return err
	}

	iopsMetrics.WithLabelValues("cas", "Exists")
	return nil
}

func (c *CAS) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	var data []byte
	err := c.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("keys"))
		if bucket == nil {
			return fmt.Errorf("%w: bucket not found", ErrNotFound)
		}

		hashBytes := bucket.Get([]byte(key))
		if hashBytes == nil {
			return fmt.Errorf("%w: key not found", ErrNotFound)
		}

		hash := hex.EncodeToString(hashBytes)
		objectPath := c.hashToObjectPath(hash)

		var err error
		data, err = os.ReadFile(objectPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%w: object file missing", ErrNotFound)
			}
			return fmt.Errorf("can't read object file: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	iopsMetrics.WithLabelValues("cas", "Get")
	return data, nil
}

func (c *CAS) Set(ctx context.Context, key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	hash := sha256.Sum256(value)
	hashHex := hex.EncodeToString(hash[:])

	objectPath := c.hashToObjectPath(hashHex)

	if _, err := os.Stat(objectPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
			return fmt.Errorf("can't create object directory: %w", err)
		}
		if err := os.WriteFile(objectPath, value, 0644); err != nil {
			return fmt.Errorf("can't write object file: %w", err)
		}
	}

	err := c.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("keys"))
		if bucket == nil {
			return fmt.Errorf("%w: bucket not found", ErrNotFound)
		}

		return bucket.Put([]byte(key), hash[:])
	})

	if err != nil {
		return err
	}

	iopsMetrics.WithLabelValues("cas", "Set")
	return nil
}

func (c *CAS) List(ctx context.Context, prefix string) ([]string, error) {
	var result []string

	err := c.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("keys"))
		if bucket == nil {
			return fmt.Errorf("%w: bucket not found", ErrNotFound)
		}

		return bucket.ForEach(func(k, v []byte) error {
			key := string(k)
			if prefix == "" || len(k) >= len(prefix) && string(k[:len(prefix)]) == prefix {
				result = append(result, key)
			}
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	iopsMetrics.WithLabelValues("cas", "List")
	return result, nil
}

func (c *CAS) hashToObjectPath(hash string) string {
	if len(hash) < 4 {
		return filepath.Join(c.objectsDir, hash)
	}

	return filepath.Join(c.objectsDir, hash[:2], hash[2:4], hash[4:])
}
