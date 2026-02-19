package models

import (
	"biyobot/mixins"
	"time"
)

type DiscordMessage struct {
	mixins.BaseModel
	Action          string    `gorm:"type:varchar(50)" json:"action"` // delete | edit
	ChannelId       string    `gorm:"type:varchar(36)" json:"channel_id"`
	UserId          string    `gorm:"type:varchar(36)" json:"user_id"`
	MessageId       string    `gorm:"type:varchar(36)" json:"message_id"`
	Content         string    `gorm:"type:text;not null" json:"content"`
	ExecuteActionOn time.Time `json:"execute_action_on"`
}
