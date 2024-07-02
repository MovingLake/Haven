package wrappers

import (
	"encoding/json"
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
	Version             uint `gorm:"primaryKey"`
	ResourceID          uint
	ReferencePayloadsID uint
	OldSchema           string
	NewSchema           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt `gorm:"index"`
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
	GetReferencePayload(id uint) (ReferencePayloads, error)
	OpenTxn() *gorm.DB
	TearDown() error
	TruncateAll() error
}

type DBImpl struct {
	conn *gorm.DB
}

func NewDB(connStr string) (DB, error) {
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return nil, err
	}

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
	d.conn.Find(r, "name = ?", resource)
	if r.ID == 0 {
		return nil, fmt.Errorf("resource not found for %s", resource)
	}
	return r, nil
}

func (d *DBImpl) TearDown() error {
	return d.conn.Migrator().DropTable(&Resource{}, &ReferencePayloads{}, &ResourceVersions{})
}

func (d *DBImpl) TruncateAll() error {
	fmt.Println("Truncating tables")
	tx := d.conn.Exec("TRUNCATE TABLE resources, reference_payloads, resource_versions;")
	fmt.Println(tx.Error)
	tx.Commit()
	return nil
}

func (d *DBImpl) GetAllResources() ([]Resource, error) {
	var resources []Resource
	d.conn.Find(&resources)
	return resources, nil
}

func (d *DBImpl) GetResourceVersions(resourceID uint) ([]ResourceVersions, error) {
	var versions []ResourceVersions
	d.conn.Find(&versions)
	str, _ := json.Marshal(versions)
	fmt.Println(string(str))
	return versions, nil
}

func (d *DBImpl) GetReferencePayload(id uint) (ReferencePayloads, error) {
	var payload ReferencePayloads
	d.conn.Find(&payload, "id = ?", id)
	return payload, nil
}
