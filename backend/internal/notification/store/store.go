package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	pb "github.com/online-judge/backend/gen/go/notification/v1"
)

// NotificationStore handles notification persistence using Redis
type NotificationStore struct {
	redis *redis.Client
}

// NewNotificationStore creates a new notification store
func NewNotificationStore(redis *redis.Client) *NotificationStore {
	return &NotificationStore{
		redis: redis,
	}
}

// Create creates a new notification
func (s *NotificationStore) Create(ctx context.Context, notification *pb.Notification) error {
	notification.CreatedAt = time.Now().Format(time.RFC3339)
	notification.Read = false

	key := "notifications:" + notification.UserId
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// Add to user's notification list (sorted by time)
	s.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: string(data),
	})

	// Set expiration for old notifications (7 days)
	s.redis.Expire(ctx, key, 7*24*60*60)

	return nil
}

// List retrieves notifications for a user
func (s *NotificationStore) List(ctx context.Context, userID string, limit int32, unreadOnly bool) ([]*pb.Notification, error) {
	key := "notifications:" + userID

	// Get notifications sorted by time (newest first)
	results, err := s.redis.ZRevRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	notifications := make([]*pb.Notification, 0, len(results))
	for _, result := range results {
		var notification pb.Notification
		if err := json.Unmarshal([]byte(result), &notification); err != nil {
			continue
		}

		if unreadOnly && notification.Read {
			continue
		}

		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationStore) MarkAsRead(ctx context.Context, userID string, notificationID string) error {
	key := "notifications:" + userID

	// Get all notifications
	results, err := s.redis.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return err
	}

	// Find and update the notification
	for _, result := range results {
		var notification pb.Notification
		if err := json.Unmarshal([]byte(result), &notification); err != nil {
			continue
		}

		if notification.Id == notificationID {
			notification.Read = true
			data, err := json.Marshal(&notification)
			if err != nil {
				return err
			}

			// Remove old and add updated
			s.redis.ZRem(ctx, key, result)
			s.redis.ZAdd(ctx, key, redis.Z{
				Score:  float64(time.Now().Unix()),
				Member: string(data),
			})
			break
		}
	}

	return nil
}

// AddSubscriber adds a user to a channel's subscriber list
func (s *NotificationStore) AddSubscriber(ctx context.Context, channel string, userID string) error {
	key := "channel:" + channel + ":subscribers"
	return s.redis.SAdd(ctx, key, userID).Err()
}

// RemoveSubscriber removes a user from a channel's subscriber list
func (s *NotificationStore) RemoveSubscriber(ctx context.Context, channel string, userID string) error {
	key := "channel:" + channel + ":subscribers"
	return s.redis.SRem(ctx, key, userID).Err()
}

// GetSubscribers gets all subscribers for a channel
func (s *NotificationStore) GetSubscribers(ctx context.Context, channel string) ([]string, error) {
	key := "channel:" + channel + ":subscribers"
	return s.redis.SMembers(ctx, key).Result()
}

// Publish publishes a message to a Redis channel
func (s *NotificationStore) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return s.redis.Publish(ctx, channel, string(data)).Err()
}