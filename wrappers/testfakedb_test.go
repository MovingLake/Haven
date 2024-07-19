package wrappers_test

import (
	"testing"

	"gorm.io/gorm"
	"movinglake.com/haven/wrappers"
)

func TestReferencePayloads(t *testing.T) {
	db := wrappers.NewTestDB()
	referencePayloads := []wrappers.ReferencePayloads{
		{
			Model:   gorm.Model{ID: 1},
			Payload: "{json-payload}",
		},
		{
			Model:   gorm.Model{ID: 2},
			Payload: "{json-payload}",
		},
	}
	for _, rp := range referencePayloads {
		db.Save(&rp, nil)
	}
	rp, err := db.GetReferencePayload(1)
	if err != nil {
		t.Fatal(err)
	}
	if rp.Payload != "{json-payload}" {
		t.Fatalf("expected {json-payload}, got %s", rp.Payload)
	}
	rp, err = db.GetReferencePayload(2)
	if err != nil {
		t.Fatal(err)
	}
	if rp.Payload != "{json-payload}" {
		t.Fatalf("expected {json-payload}, got %s", rp.Payload)
	}
}

func TestGetResource(t *testing.T) {
	db := wrappers.NewTestDB()
	resources := []wrappers.Resource{
		{
			Model:   gorm.Model{ID: 1},
			Name:    "guesty.com/api/v2/reservations",
			Schema:  "{json-schema}",
			Version: 1,
		},
		{
			Model:   gorm.Model{ID: 2},
			Name:    "guesty.com/api/v2/listings",
			Schema:  "{json-schema}",
			Version: 1,
		},
	}
	for _, r := range resources {
		db.Save(&r, nil)
	}
	r, err := db.GetResource("guesty.com/api/v2/reservations", nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "guesty.com/api/v2/reservations" {
		t.Fatalf("expected guesty.com/api/v2/reservations, got %s", r.Name)
	}
	r, err = db.GetResource("guesty.com/api/v2/listings", nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "guesty.com/api/v2/listings" {
		t.Fatalf("expected guesty.com/api/v2/listings, got %s", r.Name)
	}
	r, err = db.GetResource("guesty.com/api/v2/non-existent", nil)
	if r != nil || err != nil {
		t.Errorf("expected nil, got %v %v", r, err)
	}
}

func TestResources(t *testing.T) {
	db := wrappers.NewTestDB()
	resources := []wrappers.Resource{
		{
			Model:   gorm.Model{ID: 1},
			Name:    "guesty.com/api/v2/reservations",
			Schema:  "{json-schema}",
			Version: 1,
		},
		{
			Model:   gorm.Model{ID: 2},
			Name:    "guesty.com/api/v2/listings",
			Schema:  "{json-schema}",
			Version: 1,
		},
	}
	for _, r := range resources {
		db.Save(&r, nil)
	}
	res, err := db.GetAllResources()
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(res))
	}
	if res[0].Name != "guesty.com/api/v2/reservations" && res[0].Name != "guesty.com/api/v2/listings" {
		t.Fatalf("expected reservations or listings, got %s", res[0].Name)
	}
	if res[1].Name != "guesty.com/api/v2/listings" && res[1].Name != "guesty.com/api/v2/reservations" {
		t.Fatalf("expected reservations or listings, got %s", res[1].Name)
	}
}

func TestResourceVersions(t *testing.T) {
	db := wrappers.NewTestDB()
	one := 1
	two := 2
	resourceVersions := []wrappers.ResourceVersions{
		{
			Model:              gorm.Model{ID: 1},
			Version:            1,
			ResourceID:         1,
			ReferencePayloadID: &one,
			OldSchema:          "{json-schema}",
			NewSchema:          "{json-schema}",
		},
		{
			Model:              gorm.Model{ID: 2},
			Version:            1,
			ResourceID:         2,
			ReferencePayloadID: &two,
			OldSchema:          "{json-schema}",
			NewSchema:          "{json-schema}",
		},
	}
	for _, rv := range resourceVersions {
		db.Save(&rv, nil)
	}
	rvs, err := db.GetResourceVersions(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rvs) != 1 {
		t.Fatalf("expected 1 resource version, got %d", len(rvs))
	}
	if rvs[0].ResourceID != 1 {
		t.Fatalf("expected 1, got %d", rvs[0].ResourceID)
	}
	rvs, err = db.GetResourceVersions(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rvs) != 1 {
		t.Fatalf("expected 1 resource version, got %d", len(rvs))
	}
	if rvs[0].ResourceID != 2 {
		t.Fatalf("expected 2, got %d", rvs[0].ResourceID)
	}
}
