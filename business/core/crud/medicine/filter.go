package medicine

import (
	"fmt"
	"time"

	"github.com/EnesDemirtas/medisync/foundation/validate"
	"github.com/google/uuid"
)

// QueryFilter holds the available fields a query can be filtered on.
// We are using pointer semantics because the With API mutates the value.
type QueryFilter struct {
	ID				*uuid.UUID
	Name 			*string	`validate:"omitempty,min=3"`
	Description 	*string
	Manufacturer	*string
	Type 			*string
	Tag				*uuid.UUID
	Tags 			[]uuid.UUID
	ExpiryDate		*time.Time
}

// Validate can perform a check of tha data against the validate tags.
func (qf *QueryFilter) Validate() error {
	if err := validate.Check(qf); err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	return nil
}

// WithMedicineID sets the ID field of the QueryFilter value.
func (qf *QueryFilter) WithMedicineID(medicineID uuid.UUID) {
	qf.ID = &medicineID
}

// WithName sets the Name field of the QueryFilter value.
func (qf *QueryFilter) WithName(name string) {
	qf.Name = &name
}

// WithDescription sets the Description field of the QueryFilter value.
func (qf *QueryFilter) WithDescription(description string) {
	qf.Description = &description
}

// WithManufacturer sets the Manufacturer field of the QueryFilter value.
func (qf *QueryFilter) WithManufacturer(manufacturer string) {
	qf.Manufacturer = &manufacturer
}

// WithType sets the Type field of the QueryFilter value.
func (qf *QueryFilter) WithType(mtype string) {
	qf.Type = &mtype
}

// WithTag sets the Tag field of the QueryFilter value.
func (qf *QueryFilter) WithTag(tagID uuid.UUID) {
	qf.Tag = &tagID
}

// WithTags sets the Tags field of the QueryFilter value.
func (qf *QueryFilter) WithTags(tagIDs []uuid.UUID) {
	qf.Tags = tagIDs
}