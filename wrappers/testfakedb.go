package wrappers

import (
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

func (d *TestDB) OpenTxn() *gorm.DB {
	return nil
}

func (d *TestDB) GetResource(resource string) (*Resource, error) {
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
		if v.ResourceID != resourceID {
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

func (d *TestDB) Find(dest interface{}, optTx *gorm.DB, conds ...interface{}) error {
	if e, ok := d.Errors["Find"]; ok && e != nil {
		return e
	}
	switch dest := dest.(type) {
	case *Resource:
		r, ok := d.Resource[conds[1].(string)]
		if !ok {
			return nil
		}
		*dest = r
	case *[]Resource:
		for _, r := range d.Resource {
			*dest = append(*dest, r)
		}
	case *[]ResourceVersions:
		for _, v := range d.ResourceVersions {
			*dest = append(*dest, v)
		}
	case *ReferencePayloads:
		rp, ok := d.ReferencePayloads[conds[0].(uint)]
		if !ok {
			return nil
		}
		*dest = rp
	default:
		return nil
	}
	return nil
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

func (d *TestDB) Commit(optTx *gorm.DB) error {
	if e, ok := d.Errors["Commit"]; ok && e != nil {
		return e
	}
	return nil
}

func (d *TestDB) Rollback(optTx *gorm.DB) error {
	if e, ok := d.Errors["Rollback"]; ok && e != nil {
		return e
	}
	return nil
}
