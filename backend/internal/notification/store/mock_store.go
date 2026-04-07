package store

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	pb "github.com/online-judge/backend/gen/go/notification/v1"
)

// NotificationStoreInterface defines the interface for NotificationStore
type NotificationStoreInterface interface {
	Create(ctx context.Context, notification *pb.Notification) error
	List(ctx context.Context, userID string, limit int32, unreadOnly bool) ([]*pb.Notification, error)
	MarkAsRead(ctx context.Context, userID string, notificationID string) error
	MarkAllAsRead(ctx context.Context, userID string, filterType pb.NotificationType) (int32, error)
	GetUnreadCount(ctx context.Context, userID string) (int32, map[string]int32, error)
	PublishNotification(ctx context.Context, userID string, notification *pb.Notification) error
	AddSubscriber(ctx context.Context, channel string, userID string) error
	RemoveSubscriber(ctx context.Context, channel string, userID string) error
	GetSubscribers(ctx context.Context, channel string) ([]string, error)
	Publish(ctx context.Context, channel string, message interface{}) error
}

// MockNotificationStore is a mock implementation of NotificationStoreInterface for testing
type MockNotificationStore struct {
	mu            sync.RWMutex
	Notifications map[string][]*pb.Notification // userID -> notifications
	UnreadCounts  map[string]int32              // userID -> count
	UnreadByType  map[string]map[string]int32   // userID -> type -> count
	Subscribers   map[string]map[string]bool    // channel -> userID -> subscribed
	PublishedMsgs []interface{}                 // captured published messages

	CreateError   error
	ListError     error
	MarkReadError error
}

func NewMockNotificationStore() *MockNotificationStore {
	return &MockNotificationStore{
		Notifications: make(map[string][]*pb.Notification),
		UnreadCounts:  make(map[string]int32),
		UnreadByType:  make(map[string]map[string]int32),
		Subscribers:   make(map[string]map[string]bool),
		PublishedMsgs: make([]interface{}, 0),
	}
}

func (m *MockNotificationStore) Create(ctx context.Context, notification *pb.Notification) error {
	if m.CreateError != nil {
		return m.CreateError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if notification.CreatedAt == "" {
		notification.CreatedAt = time.Now().Format(time.RFC3339)
	}
	notification.Read = false

	m.Notifications[notification.UserId] = append(m.Notifications[notification.UserId], notification)
	m.UnreadCounts[notification.UserId]++

	if m.UnreadByType[notification.UserId] == nil {
		m.UnreadByType[notification.UserId] = make(map[string]int32)
	}
	m.UnreadByType[notification.UserId][notification.Type.String()]++

	return nil
}

func (m *MockNotificationStore) List(ctx context.Context, userID string, limit int32, unreadOnly bool) ([]*pb.Notification, error) {
	if m.ListError != nil {
		return nil, m.ListError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	notifications, ok := m.Notifications[userID]
	if !ok {
		return []*pb.Notification{}, nil
	}

	var result []*pb.Notification
	for _, n := range notifications {
		if unreadOnly && n.Read {
			continue
		}
		result = append(result, n)
		if int32(len(result)) >= limit {
			break
		}
	}

	return result, nil
}

func (m *MockNotificationStore) MarkAsRead(ctx context.Context, userID string, notificationID string) error {
	if m.MarkReadError != nil {
		return m.MarkReadError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	notifications, ok := m.Notifications[userID]
	if !ok {
		// Idempotent - if user doesn't exist, just return success
		return nil
	}

	for _, n := range notifications {
		if n.Id == notificationID && !n.Read {
			n.Read = true
			m.UnreadCounts[userID]--
			if m.UnreadByType[userID] != nil {
				m.UnreadByType[userID][n.Type.String()]--
			}
			break
		}
	}

	return nil
}

func (m *MockNotificationStore) MarkAllAsRead(ctx context.Context, userID string, filterType pb.NotificationType) (int32, error) {
	if m.MarkReadError != nil {
		return 0, m.MarkReadError
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	notifications, ok := m.Notifications[userID]
	if !ok {
		return 0, nil
	}

	var count int32
	for _, n := range notifications {
		if n.Read {
			continue
		}
		if filterType != pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED && n.Type != filterType {
			continue
		}
		n.Read = true
		count++
	}

	if count > 0 {
		m.UnreadCounts[userID] -= count
		if filterType == pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED {
			m.UnreadCounts[userID] = 0
			m.UnreadByType[userID] = make(map[string]int32)
		}
	}

	return count, nil
}

func (m *MockNotificationStore) GetUnreadCount(ctx context.Context, userID string) (int32, map[string]int32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.UnreadCounts[userID]
	countByType := make(map[string]int32)
	if byType, ok := m.UnreadByType[userID]; ok {
		for k, v := range byType {
			if v > 0 {
				countByType[k] = v
			}
		}
	}

	return total, countByType, nil
}

func (m *MockNotificationStore) PublishNotification(ctx context.Context, userID string, notification *pb.Notification) error {
	m.mu.Lock()
	m.PublishedMsgs = append(m.PublishedMsgs, notification)
	m.mu.Unlock()
	return nil
}

func (m *MockNotificationStore) AddSubscriber(ctx context.Context, channel string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Subscribers[channel] == nil {
		m.Subscribers[channel] = make(map[string]bool)
	}
	m.Subscribers[channel][userID] = true
	return nil
}

func (m *MockNotificationStore) RemoveSubscriber(ctx context.Context, channel string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Subscribers[channel] != nil {
		delete(m.Subscribers[channel], userID)
	}
	return nil
}

func (m *MockNotificationStore) GetSubscribers(ctx context.Context, channel string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var subscribers []string
	if m.Subscribers[channel] != nil {
		for userID := range m.Subscribers[channel] {
			subscribers = append(subscribers, userID)
		}
	}
	return subscribers, nil
}

func (m *MockNotificationStore) Publish(ctx context.Context, channel string, message interface{}) error {
	m.mu.Lock()
	m.PublishedMsgs = append(m.PublishedMsgs, &publishedMessage{Channel: channel, Message: message})
	m.mu.Unlock()
	return nil
}

// Helper struct to track published messages with their channels
type publishedMessage struct {
	Channel string
	Message interface{}
}

// Ensure MockNotificationStore implements NotificationStoreInterface
var _ NotificationStoreInterface = (*MockNotificationStore)(nil)

// Helper method for tests to get published messages
func (m *MockNotificationStore) GetPublishedMessages() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.PublishedMsgs
}

// Helper method for tests to reset state
func (m *MockNotificationStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Notifications = make(map[string][]*pb.Notification)
	m.UnreadCounts = make(map[string]int32)
	m.UnreadByType = make(map[string]map[string]int32)
	m.Subscribers = make(map[string]map[string]bool)
	m.PublishedMsgs = make([]interface{}, 0)
}

// Helper to create JSON of notifications (for tests that need JSON marshaling)
func notificationsToJSON(notifications []*pb.Notification) string {
	data, _ := json.Marshal(notifications)
	return string(data)
}
