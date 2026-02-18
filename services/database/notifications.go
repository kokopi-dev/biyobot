package database

import (
	"biyobot/models"
	"time"

	"gorm.io/gorm"
)

type NotificationsRepo struct {
	dbm *DatabaseManager
}

func NewNotificationsRepo(dbm *DatabaseManager) *NotificationsRepo{
	return &NotificationsRepo{dbm: dbm}
}


func (r *NotificationsRepo) GetAllNotifications() ([]models.Notification, error) {
	var notifications []models.Notification
	result := r.dbm.App().Find(&notifications)
	return notifications, result.Error
}

func (r *NotificationsRepo) DeleteNotification(notificationId string) (error) {
	result := r.dbm.App().Delete(&models.Notification{}, "id = ?", notificationId)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type AddNotificationDto struct {
	Service  string    `json:"service"`
	Metadata string    `json:"metadata"`
	NotifyAt time.Time `json:"notify_at"`
	Title    string    `json:"title"`
	Message  string    `json:"message"`
}

func (r *NotificationsRepo) AddNotification(data AddNotificationDto) (*models.Notification, error) {
	notification := &models.Notification{
		Service:  data.Service,
		Metadata: data.Metadata,
		NotifyAt: data.NotifyAt,
		Title:    data.Title,
		Message:  data.Message,
	}
	result := r.dbm.App().Create(notification)
	return notification, result.Error
}

type EditNotificationDto struct {
	ID       string    `json:"id"`
	Service  string    `json:"service"`
	Metadata string    `json:"metadata"`
	NotifyAt time.Time `json:"notify_at"`
	Title    string    `json:"title"`
	Message  string    `json:"message"`
}

func (r *NotificationsRepo) EditNotification(data EditNotificationDto) (*models.Notification, error) {
	var notification models.Notification
	if err := r.dbm.App().First(&notification, "id = ?", data.ID).Error; err != nil {
		return nil, err
	}

	result := r.dbm.App().Model(&notification).Updates(map[string]any{
		"service":   data.Service,
		"metadata":  data.Metadata,
		"notify_at": data.NotifyAt,
		"title":     data.Title,
		"message":   data.Message,
	})
	return &notification, result.Error
}
