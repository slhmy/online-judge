package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slhmy/online-judge/backend/internal/notification/store"
	"github.com/slhmy/online-judge/backend/internal/pkg/middleware"
	pb "github.com/slhmy/online-judge/gen/go/notification/v1"
)

func TestNotificationService_Integration_GetNotifications(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockNotificationStore, context.Context)
		request *pb.GetNotificationsRequest
		want    func(t *testing.T, resp *pb.GetNotificationsResponse, err error)
	}{
		{
			name: "get notifications for user",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-1",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
					Title:  "Submission Judged",
					Body:   "Your submission was accepted",
				})
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-2",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT,
					Title:  "System Alert",
					Body:   "Maintenance scheduled",
				})
			},
			request: &pb.GetNotificationsRequest{Limit: 10},
			want: func(t *testing.T, resp *pb.GetNotificationsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Notifications, 2)
			},
		},
		{
			name: "get unread only",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-1",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
					Title:  "Unread",
					Body:   "Unread notification",
				})
				// Mark one as read
				_ = m.MarkAsRead(ctx, "user-1", "notif-1")
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-2",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT,
					Title:  "Unread",
					Body:   "Another unread",
				})
			},
			request: &pb.GetNotificationsRequest{Limit: 10, UnreadOnly: true},
			want: func(t *testing.T, resp *pb.GetNotificationsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Notifications, 1)
			},
		},
		{
			name:    "get notifications for user with none",
			setup:   func(m *store.MockNotificationStore, ctx context.Context) {},
			request: &pb.GetNotificationsRequest{Limit: 10},
			want: func(t *testing.T, resp *pb.GetNotificationsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Notifications, 0)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				m.ListError = assert.AnError
			},
			request: &pb.GetNotificationsRequest{Limit: 10},
			want: func(t *testing.T, resp *pb.GetNotificationsResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockNotificationStore()
			ctx := middleware.ContextWithUserID(context.Background(), "user-1")

			tt.setup(mockStore, ctx)

			service := NewNotificationServiceWithStore(nil, mockStore)
			resp, err := service.GetNotifications(ctx, tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestNotificationService_Integration_MarkAsRead(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockNotificationStore, context.Context)
		request *pb.MarkAsReadRequest
		want    func(t *testing.T, resp *pb.MarkAsReadResponse, err error)
	}{
		{
			name: "mark notification as read",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-1",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
					Title:  "Test",
					Body:   "Test",
				})
			},
			request: &pb.MarkAsReadRequest{Id: "notif-1"},
			want: func(t *testing.T, resp *pb.MarkAsReadResponse, err error) {
				require.NoError(t, err)
				assert.True(t, resp.Success)
			},
		},
		{
			name:    "mark non-existent notification",
			setup:   func(m *store.MockNotificationStore, ctx context.Context) {},
			request: &pb.MarkAsReadRequest{Id: "non-existent"},
			want: func(t *testing.T, resp *pb.MarkAsReadResponse, err error) {
				// Should still succeed (idempotent)
				require.NoError(t, err)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				m.MarkReadError = assert.AnError
			},
			request: &pb.MarkAsReadRequest{Id: "notif-1"},
			want: func(t *testing.T, resp *pb.MarkAsReadResponse, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockNotificationStore()
			ctx := middleware.ContextWithUserID(context.Background(), "user-1")

			tt.setup(mockStore, ctx)

			service := NewNotificationServiceWithStore(nil, mockStore)
			resp, err := service.MarkAsRead(ctx, tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestNotificationService_Integration_MarkAllAsRead(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockNotificationStore, context.Context)
		request *pb.MarkAllAsReadRequest
		want    func(t *testing.T, resp *pb.MarkAllAsReadResponse, err error)
	}{
		{
			name: "mark all as read",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-1",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
					Title:  "Test 1",
					Body:   "Test",
				})
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-2",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT,
					Title:  "Test 2",
					Body:   "Test",
				})
			},
			request: &pb.MarkAllAsReadRequest{},
			want: func(t *testing.T, resp *pb.MarkAllAsReadResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, int32(2), resp.Count)
			},
		},
		{
			name: "mark all as read with type filter",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-1",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
					Title:  "Test 1",
					Body:   "Test",
				})
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-2",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT,
					Title:  "Test 2",
					Body:   "Test",
				})
			},
			request: &pb.MarkAllAsReadRequest{
				Type: pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
			},
			want: func(t *testing.T, resp *pb.MarkAllAsReadResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, int32(1), resp.Count)
			},
		},
		{
			name:    "mark all as read with no notifications",
			setup:   func(m *store.MockNotificationStore, ctx context.Context) {},
			request: &pb.MarkAllAsReadRequest{},
			want: func(t *testing.T, resp *pb.MarkAllAsReadResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, int32(0), resp.Count)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				m.MarkReadError = assert.AnError
			},
			request: &pb.MarkAllAsReadRequest{},
			want: func(t *testing.T, resp *pb.MarkAllAsReadResponse, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockNotificationStore()
			ctx := middleware.ContextWithUserID(context.Background(), "user-1")

			tt.setup(mockStore, ctx)

			service := NewNotificationServiceWithStore(nil, mockStore)
			resp, err := service.MarkAllAsRead(ctx, tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestNotificationService_Integration_GetUnreadCount(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockNotificationStore, context.Context)
		request *pb.GetUnreadCountRequest
		want    func(t *testing.T, resp *pb.GetUnreadCountResponse, err error)
	}{
		{
			name: "get unread count",
			setup: func(m *store.MockNotificationStore, ctx context.Context) {
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-1",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
					Title:  "Test 1",
					Body:   "Test",
				})
				_ = m.Create(ctx, &pb.Notification{
					Id:     "notif-2",
					UserId: "user-1",
					Type:   pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT,
					Title:  "Test 2",
					Body:   "Test",
				})
			},
			request: &pb.GetUnreadCountRequest{},
			want: func(t *testing.T, resp *pb.GetUnreadCountResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, int32(2), resp.Count)
				assert.NotEmpty(t, resp.CountByType)
			},
		},
		{
			name:    "get unread count with no notifications",
			setup:   func(m *store.MockNotificationStore, ctx context.Context) {},
			request: &pb.GetUnreadCountRequest{},
			want: func(t *testing.T, resp *pb.GetUnreadCountResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, int32(0), resp.Count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockNotificationStore()
			ctx := middleware.ContextWithUserID(context.Background(), "user-1")

			tt.setup(mockStore, ctx)

			service := NewNotificationServiceWithStore(nil, mockStore)
			resp, err := service.GetUnreadCount(ctx, tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestNotificationService_SubscribeUnsubscribe(t *testing.T) {
	mockStore := store.NewMockNotificationStore()
	service := NewNotificationServiceWithStore(nil, mockStore)
	ctx := context.Background()

	// Subscribe
	progressChan, err := service.Subscribe(ctx, "user-1", "channel-1")
	require.NoError(t, err)
	assert.NotNil(t, progressChan)

	// Verify subscriber added
	subscribers, err := mockStore.GetSubscribers(ctx, "channel-1")
	require.NoError(t, err)
	assert.Contains(t, subscribers, "user-1")

	// Unsubscribe
	err = service.Unsubscribe(ctx, "user-1", "channel-1")
	require.NoError(t, err)

	// Verify subscriber removed
	subscribers, err = mockStore.GetSubscribers(ctx, "channel-1")
	require.NoError(t, err)
	assert.NotContains(t, subscribers, "user-1")
}

func TestNotificationService_Broadcast(t *testing.T) {
	mockStore := store.NewMockNotificationStore()
	service := NewNotificationServiceWithStore(nil, mockStore)
	ctx := context.Background()

	// Subscribe to channel
	progressChan, err := service.Subscribe(ctx, "user-1", "test-channel")
	require.NoError(t, err)

	// Broadcast message
	progress := &pb.JudgeProgress{
		SubmissionId: "sub-1",
		Status:       "running",
		CurrentCase:  1,
		TotalCases:   5,
	}

	err = service.Broadcast(ctx, "test-channel", progress)
	require.NoError(t, err)

	// Receive message (non-blocking check with select)
	select {
	case msg := <-progressChan:
		assert.Equal(t, "sub-1", msg.SubmissionId)
		assert.Equal(t, int32(1), msg.CurrentCase)
	default:
		t.Log("No message received immediately (async)")
	}
}

func TestNotificationService_Integration_CreateNotification(t *testing.T) {
	mockStore := store.NewMockNotificationStore()
	service := NewNotificationServiceWithStore(nil, mockStore)
	ctx := context.Background()

	notification := &pb.Notification{
		UserId: "user-1",
		Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
		Title:  "Submission Accepted",
		Body:   "Your solution passed all test cases",
		Data: map[string]string{
			"submission_id": "sub-123",
			"verdict":       "correct",
		},
	}

	err := service.CreateNotification(ctx, notification)
	require.NoError(t, err)
	assert.NotEmpty(t, notification.Id)

	// Verify notification was stored
	notifications, err := mockStore.List(ctx, "user-1", 10, false)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, "Submission Accepted", notifications[0].Title)
}

func TestNotificationService_Integration_NotifySubmissionJudged(t *testing.T) {
	mockStore := store.NewMockNotificationStore()
	service := NewNotificationServiceWithStore(nil, mockStore)
	ctx := context.Background()

	err := service.NotifySubmissionJudged(ctx, "user-1", "sub-123", "correct")
	require.NoError(t, err)

	// Verify notification was created
	notifications, err := mockStore.List(ctx, "user-1", 10, false)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED, notifications[0].Type)
	assert.Contains(t, notifications[0].Body, "correct")
}

func TestNotificationService_Integration_NotifyContestAnnouncement(t *testing.T) {
	mockStore := store.NewMockNotificationStore()
	service := NewNotificationServiceWithStore(nil, mockStore)
	ctx := context.Background()

	recipients := []string{"user-1", "user-2", "user-3"}
	err := service.NotifyContestAnnouncement(ctx, "contest-1", "Important Update", "The contest will start in 10 minutes", recipients)
	require.NoError(t, err)

	// Verify notifications were created for all recipients
	for _, userID := range recipients {
		notifications, err := mockStore.List(ctx, userID, 10, false)
		require.NoError(t, err)
		assert.Len(t, notifications, 1)
		assert.Equal(t, pb.NotificationType_NOTIFICATION_TYPE_CONTEST_ANNOUNCEMENT, notifications[0].Type)
	}
}

func TestNotificationService_Integration_NotifySystemAlert(t *testing.T) {
	mockStore := store.NewMockNotificationStore()
	service := NewNotificationServiceWithStore(nil, mockStore)
	ctx := context.Background()

	recipients := []string{"user-1", "user-2"}
	err := service.NotifySystemAlert(ctx, "Maintenance", "System will be down for maintenance", "high", recipients)
	require.NoError(t, err)

	// Verify notifications
	notifications, err := mockStore.List(ctx, "user-1", 10, false)
	require.NoError(t, err)
	assert.Len(t, notifications, 1)
	assert.Equal(t, pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT, notifications[0].Type)
}

// Integration test: Full notification flow
func TestNotificationService_Integration_FullFlow(t *testing.T) {
	mockStore := store.NewMockNotificationStore()
	service := NewNotificationServiceWithStore(nil, mockStore)
	ctx := middleware.ContextWithUserID(context.Background(), "user-1")

	// Step 1: Create multiple notifications
	err := service.NotifySubmissionJudged(ctx, "user-1", "sub-1", "correct")
	require.NoError(t, err)

	err = service.NotifySubmissionJudged(ctx, "user-1", "sub-2", "wrong-answer")
	require.NoError(t, err)

	// Step 2: Get unread count
	countResp, err := service.GetUnreadCount(ctx, &pb.GetUnreadCountRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(2), countResp.Count)

	// Step 3: Get notifications
	listResp, err := service.GetNotifications(ctx, &pb.GetNotificationsRequest{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, listResp.Notifications, 2)

	// Step 4: Mark one as read
	_, err = service.MarkAsRead(ctx, &pb.MarkAsReadRequest{Id: listResp.Notifications[0].Id})
	require.NoError(t, err)

	// Step 5: Verify unread count decreased
	countResp, err = service.GetUnreadCount(ctx, &pb.GetUnreadCountRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), countResp.Count)

	// Step 6: Mark all as read
	_, err = service.MarkAllAsRead(ctx, &pb.MarkAllAsReadRequest{})
	require.NoError(t, err)

	// Step 7: Verify all read
	countResp, err = service.GetUnreadCount(ctx, &pb.GetUnreadCountRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(0), countResp.Count)
}
