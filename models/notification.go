package models

import (
	"biyobot/mixins"
	"time"
)

type Notification struct {
	mixins.BaseModel
	Service  string    `gorm:"type:varchar(50)" json:"service"`
	Metadata string    `gorm:"type:text;not null" json:"metadata"`
	NotifyAt time.Time `json:"notify_at"`
	Title    string    `gorm:"type:varchar(200);not null" json:"title"`
	Message  string    `gorm:"type:varchar(200);not null" json:"message"`
}
