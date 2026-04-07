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

	// Set expiration for old notifications (30 days)
	s.redis.Expire(ctx, key, 30*24*60*60*time.Second)

	// Increment unread count for the user
	s.redis.Incr(ctx, "notifications:"+notification.UserId+":unread")

	// Increment unread count by type
	typeKey := "notifications:" + notification.UserId + ":unread:" + notification.Type.String()
	s.redis.Incr(ctx, typeKey)

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

		if notification.Id == notificationID && !notification.Read {
			notification.Read = true
			data, err := json.Marshal(&notification)
			if err != nil {
				return err
			}

			// Remove old and add updated
			score, _ := s.redis.ZScore(ctx, key, result).Result()
			s.redis.ZRem(ctx, key, result)
			s.redis.ZAdd(ctx, key, redis.Z{
				Score:  score,
				Member: string(data),
			})

			// Decrement unread count
			s.redis.Decr(ctx, "notifications:"+userID+":unread")
			typeKey := "notifications:" + userID + ":unread:" + notification.Type.String()
			s.redis.Decr(ctx, typeKey)

			break
		}
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
// Optionally filters by type
func (s *NotificationStore) MarkAllAsRead(ctx context.Context, userID string, filterType pb.NotificationType) (int32, error) {
	key := "notifications:" + userID

	// Get all notifications
	results, err := s.redis.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return 0, err
	}

	var count int32
	for _, result := range results {
		var notification pb.Notification
		if err := json.Unmarshal([]byte(result), &notification); err != nil {
			continue
		}

		// Skip if already read
		if notification.Read {
			continue
		}

		// Skip if type filter doesn't match
		if filterType != pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED && notification.Type != filterType {
			continue
		}

		notification.Read = true
		data, err := json.Marshal(&notification)
		if err != nil {
			continue
		}

		// Remove old and add updated
		score, _ := s.redis.ZScore(ctx, key, result).Result()
		s.redis.ZRem(ctx, key, result)
		s.redis.ZAdd(ctx, key, redis.Z{
			Score:  score,
			Member: string(data),
		})

		count++
	}

	// Reset unread counts
	if count > 0 {
		if filterType == pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED {
			// Reset all unread counts
			s.redis.Set(ctx, "notifications:"+userID+":unread", 0, 0)
			// Reset all type counts
			for i := 1; i <= 5; i++ {
				typeKey := "notifications:" + userID + ":unread:" + pb.NotificationType(i).String()
				s.redis.Set(ctx, typeKey, 0, 0)
			}
		} else {
			// Only decrement the filtered type count
			s.redis.DecrBy(ctx, "notifications:"+userID+":unread", int64(count))
			typeKey := "notifications:" + userID + ":unread:" + filterType.String()
			s.redis.Set(ctx, typeKey, 0, 0)
		}
	}

	return count, nil
}

// GetUnreadCount gets the unread notification count for a user
func (s *NotificationStore) GetUnreadCount(ctx context.Context, userID string) (int32, map[string]int32, error) {
	// Get total unread count
	total, err := s.redis.Get(ctx, "notifications:"+userID+":unread").Int()
	if err != nil {
		if err == redis.Nil {
			total = 0
		} else {
			return 0, nil, err
		}
	}

	// Get counts by type
	countByType := make(map[string]int32)
	for i := 1; i <= 5; i++ {
		nt := pb.NotificationType(i)
		typeKey := "notifications:" + userID + ":unread:" + nt.String()
		count, err := s.redis.Get(ctx, typeKey).Int()
		if err != nil && err != redis.Nil {
			continue
		}
		if count > 0 {
			countByType[nt.String()] = int32(count)
		}
	}

	return int32(total), countByType, nil
}

// PublishNotification publishes a notification to the user's notification channel
func (s *NotificationStore) PublishNotification(ctx context.Context, userID string, notification *pb.Notification) error {
	channel := "notifications:user:" + userID
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	return s.redis.Publish(ctx, channel, string(data)).Err()
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
