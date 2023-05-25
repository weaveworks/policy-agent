package uuid

import (
	"database/sql/driver"
	satori "github.com/satori/go.uuid"
)

// Value implements the driver.Valuer interface.
func (uuid UUID) Value() (driver.Value, error) {
	return satori.UUID(uuid).Value()
}

// Scan implements the sql.Scanner interface.
func (uuid *UUID) Scan(src interface{}) error {
	id := satori.UUID(*uuid)
	idPtr := &id
	err := idPtr.Scan(src)
	if err != nil {
		return err
	}
	*uuid = UUID(id)
	return nil
}

// NullUUID can be used with the standard sql package to represent a
// UUID value that can be NULL in the database
type NullUUID struct {
	UUID  UUID
	Valid bool
}

// Value implements the driver.Valuer interface.
func (u NullUUID) Value() (driver.Value, error) {
	if !u.Valid {
		return nil, nil
	}
	// Delegate to UUID Value function
	return u.UUID.Value()
}

// Scan implements the sql.Scanner interface.
func (u *NullUUID) Scan(src interface{}) error {
	if src == nil {
		u.UUID, u.Valid = Nil, false
		return nil
	}

	// Delegate to UUID Scan function
	u.Valid = true
	return u.UUID.Scan(src)
}
