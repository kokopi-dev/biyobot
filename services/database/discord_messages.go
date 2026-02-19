package database

import (
	"biyobot/models"
	"biyobot/utils"
	"time"

	"github.com/google/uuid"
)

type DiscordMessageRepo struct {
	dbm *DatabaseManager
}

func NewDiscordMessageRepo(dbm *DatabaseManager) *DiscordMessageRepo {
	return &DiscordMessageRepo{dbm: dbm}
}

func (r *DiscordMessageRepo) GetAllExpiredMessages() ([]models.DiscordMessage, error) {
	var messages []models.DiscordMessage
	err := r.dbm.App().
		Where("execute_action_on <= ? AND action = ?", utils.JapanTimeNow(), "delete").
		Find(&messages).Error
	return messages, err
}

type AddDiscordMessageDto struct {
	Action          string
	ChannelId       string
	UserId          string
	MessageId       string
	Content         string
	ExecuteActionOn time.Time
}

func (r *DiscordMessageRepo) AddMessage(data AddDiscordMessageDto) (*models.DiscordMessage, error) {
	message := &models.DiscordMessage{
		Action:          data.Action,
		ChannelId:       data.ChannelId,
		UserId:          data.UserId,
		MessageId:       data.MessageId,
		Content:         data.Content,
		ExecuteActionOn: data.ExecuteActionOn,
	}
	err := r.dbm.App().Create(message).Error
	return message, err
}

func (r *DiscordMessageRepo) DeleteMessageBatch(ids []uuid.UUID) error {
	return r.dbm.App().
		Where("id IN ?", ids).
		Delete(&models.DiscordMessage{}).Error
}
