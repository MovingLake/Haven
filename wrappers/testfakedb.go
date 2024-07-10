package wrappers

import (
	"fmt"

	"gorm.io/gorm"
)

type TestDB struct {
	Resource          map[string]Resource
	ResourceVersions  map[uint]ResourceVersions
	ReferencePayloads map[uint]ReferencePayloads
}

func NewTestDB() DB {
	return &TestDB{
		Resource:          make(map[string]Resource),
		ResourceVersions:  make(map[uint]ResourceVersions),
		ReferencePayloads: make(map[uint]ReferencePayloads),
	}
}

func (d *TestDB) OpenTxn() *gorm.DB {
	return nil
}

func (d *TestDB) GetResource(resource string) (*Resource, error) {
	r, ok := d.Resource[resource]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resource)
	}
	return &r, nil
}

func (d *TestDB) TearDown() error {
	return nil
}

func (d *TestDB) TruncateAll() error {
	d.ReferencePayloads = make(map[uint]ReferencePayloads)
	d.Resource = make(map[string]Resource)
	d.ResourceVersions = make(map[uint]ResourceVersions)
	return nil
}

func (d *TestDB) GetAllResources() ([]Resource, error) {
	var resources []Resource
	for _, r := range d.Resource {
		resources = append(resources, r)
	}
	return resources, nil
}

func (d *TestDB) GetResourceVersions(resourceID uint) ([]ResourceVersions, error) {
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
	rp, ok := d.ReferencePayloads[id]
	if !ok {
		return nil, nil
	}
	return &rp, nil
}

func (d *TestDB) Find(dest interface{}, optTx *gorm.DB, conds ...interface{}) error {
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
	switch value := value.(type) {
	case *Resource:
		d.Resource[value.Name] = *value
	case *ResourceVersions:
		d.ResourceVersions[value.ID] = *value
	case *ReferencePayloads:
		d.ReferencePayloads[value.ID] = *value
	default:
		return nil
	}
	return nil
}

func (d *TestDB) Commit(optTx *gorm.DB) error {
	return nil
}
