package wrappers

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	_ "github.com/lib/pq"
)

// Resource table stores the schema for a resource. E.g:
// Name: guesty.com/api/v2/reservations
// Schema: {json-schema}
type Resource struct {
	gorm.Model
	Name    string `gorm:"index:idx_name,unique"`
	Schema  string
	Version uint
}

// ResourceVersions table stores how the schema has evolved over time. It also references
// the payload that triggered the new version.
type ResourceVersions struct {
	gorm.Model
	Version            uint
	ResourceID         int
	Resource           Resource `gorm:"constraint:OnDelete:CASCADE;"`
	ReferencePayloadID *int
	ReferencePayload   *ReferencePayloads `gorm:"constraint:OnDelete:SET NULL;"`
	OldSchema          string
	NewSchema          string
}

type ReferencePayloads struct {
	gorm.Model
	ResourceID int
	Resource   Resource `gorm:"constraint:OnDelete:CASCADE;"`
	Payload    string
}

type DB interface {
	GetResource(resource string, optTx *gorm.DB) (*Resource, error)
	GetAllResources() ([]Resource, error)
	GetResourceVersion(versionID uint, optTx *gorm.DB) (ResourceVersions, error)
	GetResourceVersions(resourceID uint) ([]ResourceVersions, error)
	GetReferencePayload(id uint) (*ReferencePayloads, error)
	OpenTxn() *gorm.DB
	TearDown() error
	TruncateAll() error
	Save(value interface{}, optTx *gorm.DB) error
	SelectResourceForUpdate(resourceName string, optTx *gorm.DB) (*Resource, error)
	Transaction(f func(tx *gorm.DB) error) error
}

type DBImpl struct {
	conn *gorm.DB
}

func NewDB(connStr string) (DB, error) {
	fmt.Printf("Running with DB at %s\n", connStr)
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

func (d *DBImpl) GetResource(resource string, optTx *gorm.DB) (*Resource, error) {
	r := &Resource{}
	if optTx != nil {
		ret := optTx.Find(r, "name = ?", resource)
		if ret.RowsAffected == 0 {
			return nil, nil
		}
		return r, ret.Error
	}
	ret := d.conn.Find(r, "name = ?", resource)
	if ret.RowsAffected == 0 {
		return nil, nil
	}
	return r, ret.Error
}

func (d *DBImpl) GetResourceVersion(versionID uint, optTx *gorm.DB) (ResourceVersions, error) {
	r := ResourceVersions{}
	ret := d.conn.Find(&r, "id = ?", versionID)
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

func (d *DBImpl) Save(value interface{}, optTx *gorm.DB) error {
	if optTx == nil {
		res := d.conn.Save(value)
		return res.Error
	}
	res := optTx.Save(value)
	return res.Error
}

func (d *DBImpl) SelectResourceForUpdate(resourceName string, optTx *gorm.DB) (*Resource, error) {
	r := &Resource{}
	t := optTx.Clauses(clause.Locking{
		Strength: "UPDATE",
	}).Find(r, "name = ?", resourceName)
	if t.Error != nil {
		return nil, t.Error
	}
	return r, nil
}

func (d *DBImpl) Transaction(f func(tx *gorm.DB) error) error {
	return d.conn.Transaction(f)
}
