package wrappers

import (
	"context"
	"database/sql"
	"reflect"
	"time"

	"gorm.io/gorm"
)

type TestDB struct {
	Errors            map[string]error
	IDs               map[string]uint
	Resource          map[string]Resource
	ResourceVersions  map[uint]ResourceVersions
	ReferencePayloads map[uint]ReferencePayloads
}

func NewTestDB() DB {
	return &TestDB{
		Errors: make(map[string]error),
		IDs: map[string]uint{
			"Resource":          0,
			"ResourceVersions":  0,
			"ReferencePayloads": 0,
		},
		Resource:          make(map[string]Resource),
		ResourceVersions:  make(map[uint]ResourceVersions),
		ReferencePayloads: make(map[uint]ReferencePayloads),
	}
}

type MockTxCommmiter struct{}

func (m *MockTxCommmiter) Commit() error {
	return nil
}

func (m *MockTxCommmiter) Rollback() error {
	return nil
}
func (m *MockTxCommmiter) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, nil
}
func (m *MockTxCommmiter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (m *MockTxCommmiter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (m *MockTxCommmiter) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func (d *TestDB) OpenTxn() *gorm.DB {
	t := &gorm.DB{
		Config: &gorm.Config{},
	}
	t.DisableNestedTransaction = true
	t.Statement = &gorm.Statement{}
	t.Statement.ConnPool = &MockTxCommmiter{}
	return t
}

func (d *TestDB) GetResource(resource string, optTx *gorm.DB) (*Resource, error) {
	if e, ok := d.Errors["GetResource"]; ok && e != nil {
		return nil, e
	}
	r, ok := d.Resource[resource]
	if !ok {
		return nil, nil
	}
	return &r, nil
}

func (d *TestDB) TearDown() error {
	if e, ok := d.Errors["TearDown"]; ok && e != nil {
		return e
	}
	return nil
}

func (d *TestDB) TruncateAll() error {
	if e, ok := d.Errors["TruncateAll"]; ok && e != nil {
		return e
	}
	d.IDs = map[string]uint{
		"Resource":          0,
		"ResourceVersions":  0,
		"ReferencePayloads": 0,
	}
	d.ReferencePayloads = make(map[uint]ReferencePayloads)
	d.Resource = make(map[string]Resource)
	d.ResourceVersions = make(map[uint]ResourceVersions)
	return nil
}

func (d *TestDB) GetResourceVersion(versionID uint, optTx *gorm.DB) (ResourceVersions, error) {
	if e, ok := d.Errors["GetResourceVersion"]; ok && e != nil {
		return ResourceVersions{}, e
	}
	r, ok := d.ResourceVersions[versionID]
	if !ok {
		return ResourceVersions{}, nil
	}
	return r, nil
}

func (d *TestDB) GetAllResources() ([]Resource, error) {
	if e, ok := d.Errors["GetAllResources"]; ok && e != nil {
		return nil, e
	}
	var resources []Resource
	for _, r := range d.Resource {
		resources = append(resources, r)
	}
	return resources, nil
}

func (d *TestDB) GetResourceVersions(resourceID uint) ([]ResourceVersions, error) {
	if e, ok := d.Errors["GetResourceVersions"]; ok && e != nil {
		return nil, e
	}
	var versions []ResourceVersions
	for _, v := range d.ResourceVersions {
		if v.ResourceID != int(resourceID) {
			continue
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func (d *TestDB) GetReferencePayload(id uint) (*ReferencePayloads, error) {
	if e, ok := d.Errors["GetReferencePayload"]; ok && e != nil {
		return nil, e
	}
	rp, ok := d.ReferencePayloads[id]
	if !ok {
		return nil, nil
	}
	return &rp, nil
}

func (d *TestDB) Save(value interface{}, optTx *gorm.DB) error {
	if e, ok := d.Errors["Save"]; ok && e != nil {
		return e
	}
	if value == nil || reflect.ValueOf(value).IsNil() {
		return nil
	}
	switch value := value.(type) {
	case *Resource:
		n := value.Name
		r, ok := d.Resource[n]
		if ok { // Update.
			value.ID = r.ID
			value.CreatedAt = r.CreatedAt
			value.UpdatedAt = time.Now()
		} else { // Create.
			d.IDs["Resource"] += 1
			value.ID = d.IDs["Resource"]
			value.CreatedAt = time.Now()
			value.UpdatedAt = time.Now()
		}
		d.Resource[value.Name] = *value
	case *ResourceVersions:
		if value.ID != 0 { // Update.
			r := d.ResourceVersions[value.ID]
			value.CreatedAt = r.CreatedAt
			value.UpdatedAt = time.Now()
		} else { // Create.
			d.IDs["ResourceVersions"] += 1
			value.ID = d.IDs["ResourceVersions"]
			value.CreatedAt = time.Now()
			value.UpdatedAt = time.Now()
		}
		d.ResourceVersions[value.ID] = *value
	case *ReferencePayloads:
		if value.ID != 0 { // Update.
			r := d.ReferencePayloads[value.ID]
			value.CreatedAt = r.CreatedAt
			value.UpdatedAt = time.Now()
		} else { // Create.
			d.IDs["ReferencePayloads"] += 1
			value.ID = d.IDs["ReferencePayloads"]
			value.CreatedAt = time.Now()
			value.UpdatedAt = time.Now()
		}
		d.ReferencePayloads[value.ID] = *value
	default:
		return nil
	}
	return nil
}

func (d *TestDB) SelectResourceForUpdate(resourceName string, optTx *gorm.DB) (*Resource, error) {
	if e, ok := d.Errors["SelectResourceForUpdate"]; ok && e != nil {
		return nil, e
	}
	r, ok := d.Resource[resourceName]
	if !ok {
		return &Resource{
			Model: gorm.Model{
				ID: 0,
			},
			Name:    resourceName,
			Schema:  "",
			Version: 0,
		}, nil
	}
	return &r, nil
}
func (d *TestDB) Transaction(f func(tx *gorm.DB) error) error {
	return f(d.OpenTxn())
}
