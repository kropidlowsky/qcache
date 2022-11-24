package qcache

import (
	"context"
	"encoding/json"
	"io"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/allegro/bigcache/v3"
	"gorm.io/gorm"
)

// QCache - helps to cache GORM queries
type QCache struct {
	cache      *bigcache.BigCache
	objectName string
}

type GormSelect func(dest interface{}, conds ...interface{}) *gorm.DB

// NewQCache - initializes new QCache
func NewQCache(ctx context.Context, conf bigcache.Config, objectName string, verbose bool) (*QCache, error) {
	activateLogs(verbose)
	cache, err := bigcache.New(ctx, conf)
	return &QCache{cache, objectName}, err
}

func activateLogs(verbose bool) {
	if !verbose {
		log.SetOutput(io.Discard)
	} else {
		log.SetOutput(os.Stdout)
	}
}

// Find - looks for the record (based on primaryKey and dest) in the QCache first.
// If it was not found in the QCache it checks the database.
func (q *QCache) Find(gormSelect GormSelect, dest interface{}, primaryKey string) error {
	value, err := q.cache.Get(primaryKey)
	castBytesToStruct(value, &dest)
	if err == nil {
		log.Printf("Found %s with primaryKey = %s in the QCache", q.objectName, primaryKey)
		return nil
	}

	return q.checkDB(gormSelect, dest, primaryKey)
}

// checkDB - checks database using gormSelect function, dest (interface corresponds to entity) and its primaryKey
func (q *QCache) checkDB(gormSelect GormSelect, dest interface{}, primaryKey string) error {
	response := gormSelect(&dest, primaryKey)
	if response.Error != nil {
		log.Errorf("Could not find %s with primaryKey = %s in the database. Error: %s", q.objectName, primaryKey, response.Error.Error())
		return response.Error
	}
	log.Printf("Found %s with primaryKey = %s in the database", q.objectName, primaryKey)

	respBytes, err := json.Marshal(dest)
	if err != nil {
		log.Errorf("Could not marshal %s, Error: %s", q.objectName, err.Error())
	}
	q.add(primaryKey, respBytes)

	return nil
}

func (q *QCache) add(key string, value []byte) {
	q.cache.Set(key, value)
	log.Printf("Added %s with primaryKey = %s to the QCache", q.objectName, key)
}

func castBytesToStruct(b []byte, s interface{}) error {
	return json.Unmarshal(b, &s)
}
