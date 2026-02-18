package mixins

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type BaseModel struct {
	ID        uuid.UUID  `gorm:"type:varchar(36);primaryKey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		if newId, err := uuid.NewV7(); err == nil {
			b.ID = newId
		} else {
			b.ID = uuid.New()
		}
	}
	return nil
}
