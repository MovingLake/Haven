package wrappers

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/lib/pq"
)

// Resource table stores the schema for a resource. E.g:
// Name: guesty.com/api/v2/reservations
// Schema: {json-schema}
type Resource struct {
	gorm.Model
	Name    string
	Schema  string
	Version uint
}

// ResourceVersions table stores how the schema has evolved over time. It also references
// the payload that triggered the new version.
type ResourceVersions struct {
	gorm.Model
	Version             uint
	ResourceID          uint
	ReferencePayloadsID uint
	OldSchema           string
	NewSchema           string
}

type ReferencePayloads struct {
	gorm.Model
	ResourceID uint
	Payload    string
}

type DB interface {
	GetResource(resource string) (*Resource, error)
	GetAllResources() ([]Resource, error)
	GetResourceVersions(resourceID uint) ([]ResourceVersions, error)
	GetReferencePayload(id uint) (*ReferencePayloads, error)
	OpenTxn() *gorm.DB
	TearDown() error
	TruncateAll() error
	Find(dest interface{}, optTx *gorm.DB, conds ...interface{}) error
	Save(value interface{}, optTx *gorm.DB) error
	Commit(optTx *gorm.DB) error
}

type DBImpl struct {
	conn *gorm.DB
}

func NewDB(connStr string) (DB, error) {
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error getting db connection: %w", err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(5)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(10)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Migrate the schema
	db.AutoMigrate(&Resource{})
	db.AutoMigrate(&ReferencePayloads{})
	db.AutoMigrate(&ResourceVersions{})

	return &DBImpl{
		conn: db,
	}, nil
}

func (d *DBImpl) OpenTxn() *gorm.DB {
	return d.conn.Begin()
}

func (d *DBImpl) GetResource(resource string) (*Resource, error) {
	r := &Resource{}
	ret := d.conn.Find(r, "name = ?", resource)
	if r.ID == 0 {
		return nil, fmt.Errorf("resource not found for %s", resource)
	}
	return r, ret.Error
}

func (d *DBImpl) TearDown() error {
	return d.conn.Migrator().DropTable(&Resource{}, &ReferencePayloads{}, &ResourceVersions{})
}

func (d *DBImpl) TruncateAll() error {
	fmt.Println("Truncating tables")
	tx := d.conn.Exec("TRUNCATE TABLE resources, reference_payloads, resource_versions;")
	fmt.Println(tx.Error)
	return tx.Commit().Error
}

func (d *DBImpl) GetAllResources() ([]Resource, error) {
	var resources []Resource
	ret := d.conn.Find(&resources)
	return resources, ret.Error
}

func (d *DBImpl) GetResourceVersions(resourceID uint) ([]ResourceVersions, error) {
	var versions []ResourceVersions
	ret := d.conn.Find(&versions, "resource_id = ?", resourceID)
	return versions, ret.Error
}

func (d *DBImpl) GetReferencePayload(id uint) (*ReferencePayloads, error) {
	payload := &ReferencePayloads{}
	ret := d.conn.Find(payload, "id = ?", id)
	return payload, ret.Error
}

func (d *DBImpl) Find(dest interface{}, optTx *gorm.DB, conds ...interface{}) error {
	if optTx == nil {
		res := d.conn.Find(dest, conds...)
		return res.Error
	}
	res := optTx.Find(dest, conds...)
	return res.Error
}

func (d *DBImpl) Save(value interface{}, optTx *gorm.DB) error {
	if optTx == nil {
		res := d.conn.Save(value)
		return res.Error
	}
	res := optTx.Save(value)
	return res.Error
}

func (d *DBImpl) Commit(optTx *gorm.DB) error {
	if optTx == nil {
		res := d.conn.Commit()
		return res.Error
	}
	res := optTx.Commit()
	return res.Error
}
