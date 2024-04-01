package tag

import (
	"fmt"

	"github.com/EnesDemirtas/medisync/foundation/validate"
	"github.com/google/uuid"
)

// QueryFilter holds the available fields a query can be filtered on.
// We are using pointer semantics because the With API mutates the value.
type QueryFilter struct {
	ID	 *uuid.UUID
	Name *string `validate:"omitempty,min=3"`
}

// Validate can perform a check of tha data against the validate tags.
func (qf *QueryFilter) Validate() error {
	if err := validate.Check(qf); err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	return nil
}

// WithID sets the ID field of the QueryFilter value.
func (qf *QueryFilter) WithID(id uuid.UUID) {
	qf.ID = &id
}


// WithName sets the Name field of the QueryFilter value.
func (qf *QueryFilter) WithName(name string) {
	qf.Name = &name
}