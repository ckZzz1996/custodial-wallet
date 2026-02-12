package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"time"

	"custodial-wallet/pkg/logger"

	"gorm.io/gorm"
)

// Repository 通知仓储接口
type Repository interface {
	CreateNotification(n *Notification) error
	GetNotification(id uint) (*Notification, error)
	ListNotifications(userID uint, page, pageSize int) ([]*Notification, int64, error)
	ListPendingNotifications(limit int) ([]*Notification, error)
	UpdateNotification(n *Notification) error
	MarkAsRead(id uint) error
	MarkAllAsRead(userID uint) error
	CountUnread(userID uint) (int64, error)

	GetTemplate(nType NotificationType, channel Channel) (*NotificationTemplate, error)
	CreateTemplate(t *NotificationTemplate) error
	UpdateTemplate(t *NotificationTemplate) error

	GetUserSetting(userID uint, nType NotificationType) (*UserNotificationSetting, error)
	CreateUserSetting(s *UserNotificationSetting) error
	UpdateUserSetting(s *UserNotificationSetting) error
	ListUserSettings(userID uint) ([]*UserNotificationSetting, error)

	GetWebhookConfig(id uint) (*WebhookConfig, error)
	ListUserWebhooks(userID uint) ([]*WebhookConfig, error)
	CreateWebhook(w *WebhookConfig) error
	UpdateWebhook(w *WebhookConfig) error
	DeleteWebhook(id uint) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建通知仓储
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// 实现Repository接口方法...
func (r *repository) CreateNotification(n *Notification) error {
	return r.db.Create(n).Error
}

func (r *repository) GetNotification(id uint) (*Notification, error) {
	var n Notification
	if err := r.db.First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *repository) ListNotifications(userID uint, page, pageSize int) ([]*Notification, int64, error) {
	var notifications []*Notification
	var total int64

	r.db.Model(&Notification{}).Where("user_id = ?", userID).Count(&total)

	offset := (page - 1) * pageSize
	if err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

func (r *repository) ListPendingNotifications(limit int) ([]*Notification, error) {
	var notifications []*Notification
	if err := r.db.Where("status = 0 AND retry_count < 3").
		Order("created_at ASC").Limit(limit).Find(&notifications).Error; err != nil {
		return nil, err
	}
	return notifications, nil
}

func (r *repository) UpdateNotification(n *Notification) error {
	return r.db.Save(n).Error
}

func (r *repository) MarkAsRead(id uint) error {
	now := time.Now()
	return r.db.Model(&Notification{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":  3,
		"read_at": &now,
	}).Error
}

func (r *repository) MarkAllAsRead(userID uint) error {
	now := time.Now()
	return r.db.Model(&Notification{}).Where("user_id = ? AND status != 3", userID).Updates(map[string]interface{}{
		"status":  3,
		"read_at": &now,
	}).Error
}

func (r *repository) CountUnread(userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&Notification{}).Where("user_id = ? AND status != 3", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) GetTemplate(nType NotificationType, channel Channel) (*NotificationTemplate, error) {
	var t NotificationTemplate
	if err := r.db.Where("type = ? AND channel = ?", nType, channel).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *repository) CreateTemplate(t *NotificationTemplate) error {
	return r.db.Create(t).Error
}

func (r *repository) UpdateTemplate(t *NotificationTemplate) error {
	return r.db.Save(t).Error
}

func (r *repository) GetUserSetting(userID uint, nType NotificationType) (*UserNotificationSetting, error) {
	var s UserNotificationSetting
	if err := r.db.Where("user_id = ? AND type = ?", userID, nType).First(&s).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *repository) CreateUserSetting(s *UserNotificationSetting) error {
	return r.db.Create(s).Error
}

func (r *repository) UpdateUserSetting(s *UserNotificationSetting) error {
	return r.db.Save(s).Error
}

func (r *repository) ListUserSettings(userID uint) ([]*UserNotificationSetting, error) {
	var settings []*UserNotificationSetting
	if err := r.db.Where("user_id = ?", userID).Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *repository) GetWebhookConfig(id uint) (*WebhookConfig, error) {
	var w WebhookConfig
	if err := r.db.First(&w, id).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *repository) ListUserWebhooks(userID uint) ([]*WebhookConfig, error) {
	var webhooks []*WebhookConfig
	if err := r.db.Where("user_id = ?", userID).Find(&webhooks).Error; err != nil {
		return nil, err
	}
	return webhooks, nil
}

func (r *repository) CreateWebhook(w *WebhookConfig) error {
	return r.db.Create(w).Error
}

func (r *repository) UpdateWebhook(w *WebhookConfig) error {
	return r.db.Save(w).Error
}

func (r *repository) DeleteWebhook(id uint) error {
	return r.db.Delete(&WebhookConfig{}, id).Error
}

// Service 通知服务接口
type Service interface {
	Send(userID uint, nType NotificationType, data map[string]interface{}) error
	SendEmail(to, subject, content string) error
	SendSMS(phone, content string) error
	SendWebhook(userID uint, event string, data interface{}) error

	GetNotifications(userID uint, page, pageSize int) ([]*Notification, int64, error)
	MarkAsRead(userID uint, notificationID uint) error
	MarkAllAsRead(userID uint) error
	GetUnreadCount(userID uint) (int64, error)

	UpdateUserSetting(userID uint, nType NotificationType, setting *UserNotificationSetting) error
	GetUserSettings(userID uint) ([]*UserNotificationSetting, error)

	ProcessPendingNotifications() error
}

type service struct {
	repo Repository
}

// NewService 创建通知服务
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Send 发送通知
func (s *service) Send(userID uint, nType NotificationType, data map[string]interface{}) error {
	// 获取用户设置
	setting, _ := s.repo.GetUserSetting(userID, nType)

	channels := []Channel{ChannelInApp} // 默认站内通知
	if setting != nil {
		if setting.Email {
			channels = append(channels, ChannelEmail)
		}
		if setting.SMS {
			channels = append(channels, ChannelSMS)
		}
	}

	for _, channel := range channels {
		// 获取模板
		tmpl, err := s.repo.GetTemplate(nType, channel)
		if err != nil || tmpl == nil {
			continue
		}

		// 渲染内容
		title, content := s.renderTemplate(tmpl, data)

		// 创建通知
		notification := &Notification{
			UserID:  userID,
			Type:    nType,
			Channel: channel,
			Title:   title,
			Content: content,
			Status:  0,
		}

		if dataJSON, err := json.Marshal(data); err == nil {
			notification.Data = string(dataJSON)
		}

		if err := s.repo.CreateNotification(notification); err != nil {
			logger.Errorf("Failed to create notification: %v", err)
		}
	}

	return nil
}

func (s *service) renderTemplate(tmpl *NotificationTemplate, data map[string]interface{}) (string, string) {
	titleTmpl, err := template.New("title").Parse(tmpl.Title)
	if err != nil {
		return tmpl.Title, tmpl.Content
	}

	contentTmpl, err := template.New("content").Parse(tmpl.Content)
	if err != nil {
		return tmpl.Title, tmpl.Content
	}

	var titleBuf, contentBuf bytes.Buffer
	_ = titleTmpl.Execute(&titleBuf, data)
	_ = contentTmpl.Execute(&contentBuf, data)

	return titleBuf.String(), contentBuf.String()
}

// SendEmail 发送邮件
func (s *service) SendEmail(to, subject, content string) error {
	// TODO: 实现邮件发送
	logger.Infof("Sending email to %s: %s", to, subject)
	return nil
}

// SendSMS 发送短信
func (s *service) SendSMS(phone, content string) error {
	// TODO: 实现短信发送
	logger.Infof("Sending SMS to %s", phone)
	return nil
}

// SendWebhook 发送Webhook
func (s *service) SendWebhook(userID uint, event string, data interface{}) error {
	webhooks, err := s.repo.ListUserWebhooks(userID)
	if err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"event":     event,
		"data":      data,
		"timestamp": time.Now().Unix(),
	})

	for _, webhook := range webhooks {
		if webhook.Status != 1 {
			continue
		}

		// 检查事件是否匹配
		var events []string
		if err := json.Unmarshal([]byte(webhook.Events), &events); err == nil {
			matched := false
			for _, e := range events {
				if e == event || e == "*" {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		go s.sendWebhookRequest(webhook, payload)
	}

	return nil
}

func (s *service) sendWebhookRequest(webhook *WebhookConfig, payload []byte) {
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(payload))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if webhook.Secret != "" {
		// TODO: 添加签名
	}

	// 添加自定义头
	if webhook.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(webhook.Headers), &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("Webhook request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	logger.Infof("Webhook sent to %s, status: %d", webhook.URL, resp.StatusCode)
}

// GetNotifications 获取通知列表
func (s *service) GetNotifications(userID uint, page, pageSize int) ([]*Notification, int64, error) {
	return s.repo.ListNotifications(userID, page, pageSize)
}

// MarkAsRead 标记已读
func (s *service) MarkAsRead(userID uint, notificationID uint) error {
	n, err := s.repo.GetNotification(notificationID)
	if err != nil {
		return err
	}
	if n.UserID != userID {
		return errors.New("notification does not belong to user")
	}
	return s.repo.MarkAsRead(notificationID)
}

// MarkAllAsRead 标记全部已读
func (s *service) MarkAllAsRead(userID uint) error {
	return s.repo.MarkAllAsRead(userID)
}

// GetUnreadCount 获取未读数量
func (s *service) GetUnreadCount(userID uint) (int64, error) {
	return s.repo.CountUnread(userID)
}

// UpdateUserSetting 更新用户设置
func (s *service) UpdateUserSetting(userID uint, nType NotificationType, setting *UserNotificationSetting) error {
	existing, _ := s.repo.GetUserSetting(userID, nType)
	if existing != nil {
		existing.Email = setting.Email
		existing.SMS = setting.SMS
		existing.InApp = setting.InApp
		existing.Push = setting.Push
		return s.repo.UpdateUserSetting(existing)
	}

	setting.UserID = userID
	setting.Type = nType
	return s.repo.CreateUserSetting(setting)
}

// GetUserSettings 获取用户设置
func (s *service) GetUserSettings(userID uint) ([]*UserNotificationSetting, error) {
	return s.repo.ListUserSettings(userID)
}

// ProcessPendingNotifications 处理待发送的通知
func (s *service) ProcessPendingNotifications() error {
	notifications, err := s.repo.ListPendingNotifications(100)
	if err != nil {
		return err
	}

	for _, n := range notifications {
		var sendErr error

		switch n.Channel {
		case ChannelEmail:
			// TODO: 获取用户邮箱并发送
			sendErr = s.SendEmail("", n.Title, n.Content)
		case ChannelSMS:
			// TODO: 获取用户手机并发送
			sendErr = s.SendSMS("", n.Content)
		case ChannelInApp:
			// 站内通知直接标记为已发送
			sendErr = nil
		}

		now := time.Now()
		if sendErr != nil {
			n.RetryCount++
			n.ErrorMsg = sendErr.Error()
			if n.RetryCount >= 3 {
				n.Status = 2 // failed
			}
		} else {
			n.Status = 1 // sent
			n.SendAt = &now
		}

		_ = s.repo.UpdateNotification(n)
	}

	return nil
}
