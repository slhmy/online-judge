package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	pb "github.com/online-judge/gen/go/notification/v1"
	"github.com/online-judge/backend/internal/pkg/middleware"
)

func setupTestService(t *testing.T) (*NotificationService, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewNotificationService(rdb)
	return service, mr
}

func TestNotificationService_GetNotifications(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := middleware.ContextWithUserID(context.Background(), "user-1")

	// Create some test notifications
	notifications := []*pb.Notification{
		{
			Id:     "notif-1",
			UserId: "user-1",
			Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
			Title:  "Test 1",
			Body:   "Body 1",
			Read:   false,
		},
		{
			Id:     "notif-2",
			UserId: "user-1",
			Type:   pb.NotificationType_NOTIFICATION_TYPE_CONTEST_ANNOUNCEMENT,
			Title:  "Test 2",
			Body:   "Body 2",
			Read:   true,
		},
	}

	// Add notifications to Redis directly
	key := "notifications:user-1"
	for _, n := range notifications {
		data, _ := json.Marshal(n)
		_, _ = mr.ZAdd(key, float64(time.Now().Unix()), string(data))
	}

	// Test GetNotifications
	resp, err := service.GetNotifications(ctx, &pb.GetNotificationsRequest{
		Limit:      10,
		UnreadOnly: false,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Notifications, 2)

	// Test unread only
	resp, err = service.GetNotifications(ctx, &pb.GetNotificationsRequest{
		Limit:      10,
		UnreadOnly: true,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Notifications, 1)
	assert.Equal(t, "notif-1", resp.Notifications[0].Id)
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := middleware.ContextWithUserID(context.Background(), "user-1")

	// Create a test notification
	notification := &pb.Notification{
		Id:     "notif-1",
		UserId: "user-1",
		Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
		Title:  "Test",
		Body:   "Body",
		Read:   false,
	}

	// Add notification to Redis
	key := "notifications:user-1"
	data, _ := json.Marshal(notification)
	_, _ = mr.ZAdd(key, float64(time.Now().Unix()), string(data))
	_ = mr.Set("notifications:user-1:unread", "1")

	// Mark as read
	resp, err := service.MarkAsRead(ctx, &pb.MarkAsReadRequest{
		Id: "notif-1",
	})
	assert.NoError(t, err)
	assert.True(t, resp.Success)

	// Verify notification is now read by checking unread-only list
	respList, err := service.GetNotifications(ctx, &pb.GetNotificationsRequest{
		Limit:      10,
		UnreadOnly: true,
	})
	assert.NoError(t, err)
	assert.Len(t, respList.Notifications, 0)
}

func TestNotificationService_MarkAllAsRead(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := middleware.ContextWithUserID(context.Background(), "user-1")

	// Create test notifications
	notifications := []*pb.Notification{
		{
			Id:     "notif-1",
			UserId: "user-1",
			Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
			Title:  "Test 1",
			Body:   "Body 1",
			Read:   false,
		},
		{
			Id:     "notif-2",
			UserId: "user-1",
			Type:   pb.NotificationType_NOTIFICATION_TYPE_CONTEST_ANNOUNCEMENT,
			Title:  "Test 2",
			Body:   "Body 2",
			Read:   false,
		},
	}

	// Add notifications to Redis
	key := "notifications:user-1"
	for _, n := range notifications {
		data, _ := json.Marshal(n)
		_, _ = mr.ZAdd(key, float64(time.Now().Unix()), string(data))
	}
	_ = mr.Set("notifications:user-1:unread", "2")

	// Mark all as read
	resp, err := service.MarkAllAsRead(ctx, &pb.MarkAllAsReadRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.Count)

	// Verify all notifications are read
	respList, err := service.GetNotifications(ctx, &pb.GetNotificationsRequest{
		Limit:      10,
		UnreadOnly: true,
	})
	assert.NoError(t, err)
	assert.Len(t, respList.Notifications, 0)
}

func TestNotificationService_GetUnreadCount(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := middleware.ContextWithUserID(context.Background(), "user-1")

	// Set unread counts
	_ = mr.Set("notifications:user-1:unread", "5")
	_ = mr.Set("notifications:user-1:unread:NOTIFICATION_TYPE_SUBMISSION_JUDGED", "3")
	_ = mr.Set("notifications:user-1:unread:NOTIFICATION_TYPE_CONTEST_ANNOUNCEMENT", "2")

	// Get unread count
	resp, err := service.GetUnreadCount(ctx, &pb.GetUnreadCountRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(5), resp.Count)
	assert.Equal(t, int32(3), resp.CountByType["NOTIFICATION_TYPE_SUBMISSION_JUDGED"])
	assert.Equal(t, int32(2), resp.CountByType["NOTIFICATION_TYPE_CONTEST_ANNOUNCEMENT"])
}

func TestNotificationService_CreateNotification(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := context.Background()

	// Create notification
	notification := &pb.Notification{
		UserId: "user-1",
		Type:   pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT,
		Title:  "System Alert",
		Body:   "This is a test alert",
		Data: map[string]string{
			"severity": "high",
		},
	}

	err := service.CreateNotification(ctx, notification)
	assert.NoError(t, err)
	assert.NotEmpty(t, notification.Id)

	// Verify notification was created by checking via GetNotifications
	ctxWithUser := middleware.ContextWithUserID(context.Background(), "user-1")
	resp, err := service.GetNotifications(ctxWithUser, &pb.GetNotificationsRequest{
		Limit:      10,
		UnreadOnly: false,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Notifications, 1)
	assert.Equal(t, notification.Title, resp.Notifications[0].Title)
	assert.Equal(t, notification.Body, resp.Notifications[0].Body)
	assert.False(t, resp.Notifications[0].Read)
}

func TestNotificationService_NotifySubmissionJudged(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := context.Background()

	err := service.NotifySubmissionJudged(ctx, "user-1", "submission-123", "correct")
	assert.NoError(t, err)

	// Verify notification was created via GetNotifications
	ctxWithUser := middleware.ContextWithUserID(context.Background(), "user-1")
	resp, err := service.GetNotifications(ctxWithUser, &pb.GetNotificationsRequest{
		Limit:      10,
		UnreadOnly: false,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Notifications, 1)
	assert.Equal(t, pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED, resp.Notifications[0].Type)
	assert.Contains(t, resp.Notifications[0].Body, "correct")
	assert.Equal(t, "submission-123", resp.Notifications[0].Data["submission_id"])
}

func TestNotificationService_NotifyContestAnnouncement(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := context.Background()

	recipients := []string{"user-1", "user-2", "user-3"}
	err := service.NotifyContestAnnouncement(ctx, "contest-1", "Important Update", "The contest rules have changed", recipients)
	assert.NoError(t, err)

	// Verify notifications were created for all recipients
	for _, userID := range recipients {
		ctxWithUser := middleware.ContextWithUserID(context.Background(), userID)
		resp, err := service.GetNotifications(ctxWithUser, &pb.GetNotificationsRequest{
			Limit:      10,
			UnreadOnly: false,
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Notifications, 1)
		assert.Equal(t, pb.NotificationType_NOTIFICATION_TYPE_CONTEST_ANNOUNCEMENT, resp.Notifications[0].Type)
		assert.Equal(t, "contest-1", resp.Notifications[0].Data["contest_id"])
	}
}

func TestNotificationService_NotifySystemAlert(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := context.Background()

	recipients := []string{"user-1", "user-2"}
	err := service.NotifySystemAlert(ctx, "Maintenance Notice", "System will be down for maintenance", "medium", recipients)
	assert.NoError(t, err)

	// Verify notifications were created
	for _, userID := range recipients {
		ctxWithUser := middleware.ContextWithUserID(context.Background(), userID)
		resp, err := service.GetNotifications(ctxWithUser, &pb.GetNotificationsRequest{
			Limit:      10,
			UnreadOnly: false,
		})
		assert.NoError(t, err)
		assert.Len(t, resp.Notifications, 1)
		assert.Equal(t, pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT, resp.Notifications[0].Type)
		assert.Equal(t, "medium", resp.Notifications[0].Data["severity"])
	}
}

func TestNotificationService_NotifyContestStarting(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := context.Background()

	recipients := []string{"user-1"}
	err := service.NotifyContestStarting(ctx, "contest-1", "Algorithm Challenge", 15, recipients)
	assert.NoError(t, err)

	// Verify notification was created
	ctxWithUser := middleware.ContextWithUserID(context.Background(), "user-1")
	resp, err := service.GetNotifications(ctxWithUser, &pb.GetNotificationsRequest{
		Limit:      10,
		UnreadOnly: false,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Notifications, 1)
	assert.Equal(t, pb.NotificationType_NOTIFICATION_TYPE_CONTEST_STARTING, resp.Notifications[0].Type)
	assert.Contains(t, resp.Notifications[0].Body, "15 minutes")
}

func TestNotificationService_PublishProgress(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := context.Background()

	progress := &pb.JudgeProgress{
		SubmissionId: "submission-123",
		Status:       "running",
		Progress:     50,
		CurrentCase:  5,
		TotalCases:   10,
		Verdict:      "correct",
		Runtime:      1.5,
		Memory:       1024,
	}

	err := service.PublishProgress(ctx, progress)
	assert.NoError(t, err)

	// Note: miniredis doesn't fully support pub/sub, but we can check the method didn't error
}

func TestNotificationService_Unauthenticated(t *testing.T) {
	service, mr := setupTestService(t)
	defer mr.Close()

	ctx := context.Background() // No user_id

	_, err := service.GetNotifications(ctx, &pb.GetNotificationsRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")

	_, err = service.MarkAsRead(ctx, &pb.MarkAsReadRequest{Id: "notif-1"})
	assert.Error(t, err)

	_, err = service.MarkAllAsRead(ctx, &pb.MarkAllAsReadRequest{})
	assert.Error(t, err)

	_, err = service.GetUnreadCount(ctx, &pb.GetUnreadCountRequest{})
	assert.Error(t, err)
}
