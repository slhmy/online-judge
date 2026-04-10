package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	pb "github.com/online-judge/gen/go/notification/v1"
	"github.com/online-judge/backend/internal/notification/store"
	"github.com/online-judge/backend/internal/pkg/middleware"
)

// NotificationService implements the notification gRPC service
type NotificationService struct {
	pb.UnimplementedNotificationServiceServer
	redis *redis.Client
	store NotificationStoreInterface

	// Local subscriber management for judging progress updates
	// channel -> userID -> progressChan
	progressSubscribers map[string]map[string]chan *pb.JudgeProgress
	progressMu          sync.RWMutex

	// Stream subscribers for SSE notification streaming
	// userID -> notificationChan
	streamSubscribers map[string]chan *pb.Notification
	streamMu          sync.RWMutex
}

// NotificationStoreInterface defines the interface for notification store
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

// NewNotificationService creates a new notification service
func NewNotificationService(redis *redis.Client) *NotificationService {
	return NewNotificationServiceWithStore(redis, store.NewNotificationStore(redis))
}

// NewNotificationServiceWithStore creates a new notification service with a custom store (for testing)
func NewNotificationServiceWithStore(redis *redis.Client, s NotificationStoreInterface) *NotificationService {
	svc := &NotificationService{
		redis:               redis,
		store:               s,
		progressSubscribers: make(map[string]map[string]chan *pb.JudgeProgress),
		streamSubscribers:   make(map[string]chan *pb.Notification),
	}

	// Start Redis pub/sub listener for cross-instance communication
	if redis != nil {
		go svc.listenToRedisPubSub()
		go svc.listenToNotificationChannels()
	}

	return svc
}

// GetNotifications retrieves notifications for a user
func (s *NotificationService) GetNotifications(ctx context.Context, req *pb.GetNotificationsRequest) (*pb.GetNotificationsResponse, error) {
	// Get user ID from context (extracted from JWT token in interceptor)
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}

	notifications, err := s.store.List(ctx, userID, limit, req.UnreadOnly)
	if err != nil {
		return nil, err
	}

	return &pb.GetNotificationsResponse{
		Notifications: notifications,
	}, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	err := s.store.MarkAsRead(ctx, userID, req.Id)
	if err != nil {
		return &pb.MarkAsReadResponse{Success: false}, err
	}

	return &pb.MarkAsReadResponse{Success: true}, nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, req *pb.MarkAllAsReadRequest) (*pb.MarkAllAsReadResponse, error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	count, err := s.store.MarkAllAsRead(ctx, userID, req.Type)
	if err != nil {
		return nil, err
	}

	return &pb.MarkAllAsReadResponse{Count: count}, nil
}

// GetUnreadCount gets the unread notification count
func (s *NotificationService) GetUnreadCount(ctx context.Context, req *pb.GetUnreadCountRequest) (*pb.GetUnreadCountResponse, error) {
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	total, countByType, err := s.store.GetUnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &pb.GetUnreadCountResponse{
		Count:       total,
		CountByType: countByType,
	}, nil
}

// StreamNotifications streams notifications via SSE for a user
func (s *NotificationService) StreamNotifications(req *pb.StreamNotificationsRequest, stream pb.NotificationService_StreamNotificationsServer) error {
	userID := stream.Context().Value("user_id")
	if userID == nil {
		return fmt.Errorf("user not authenticated")
	}

	userIDStr := userID.(string)

	// Register stream subscriber
	s.streamMu.Lock()
	notifChan := make(chan *pb.Notification, 100)
	s.streamSubscribers[userIDStr] = notifChan
	s.streamMu.Unlock()

	log.Printf("SSE stream started for user %s", userIDStr)

	defer func() {
		s.streamMu.Lock()
		delete(s.streamSubscribers, userIDStr)
		close(notifChan)
		s.streamMu.Unlock()
		log.Printf("SSE stream ended for user %s", userIDStr)
	}()

	// Send initial unread count
	total, _, _ := s.store.GetUnreadCount(stream.Context(), userIDStr)
	if err := stream.Send(&pb.Notification{
		Id:    "system:unread-count",
		Type:  pb.NotificationType_NOTIFICATION_TYPE_UNSPECIFIED,
		Title: "Unread Count",
		Body:  fmt.Sprintf("%d", total),
		Data:  map[string]string{"total": fmt.Sprintf("%d", total)},
	}); err != nil {
		log.Printf("Failed to send initial unread count: %v", err)
	}

	// Stream notifications
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case notif := <-notifChan:
			if err := stream.Send(notif); err != nil {
				return err
			}
		}
	}
}

// Subscribe subscribes a user to a notification channel
// Returns a channel that will receive progress updates
func (s *NotificationService) Subscribe(ctx context.Context, userID string, channel string) (chan *pb.JudgeProgress, error) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()

	// Add to Redis subscriber list for cross-instance communication
	if err := s.store.AddSubscriber(ctx, channel, userID); err != nil {
		log.Printf("Failed to add subscriber to Redis: %v", err)
	}

	// Create local channel for this user
	if s.progressSubscribers[channel] == nil {
		s.progressSubscribers[channel] = make(map[string]chan *pb.JudgeProgress)
	}

	progressChan := make(chan *pb.JudgeProgress, 100)
	s.progressSubscribers[channel][userID] = progressChan

	log.Printf("User %s subscribed to channel %s", userID, channel)

	return progressChan, nil
}

// Unsubscribe removes a user from a notification channel
func (s *NotificationService) Unsubscribe(ctx context.Context, userID string, channel string) error {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()

	// Remove from Redis subscriber list
	if err := s.store.RemoveSubscriber(ctx, channel, userID); err != nil {
		log.Printf("Failed to remove subscriber from Redis: %v", err)
	}

	// Close and remove local channel
	if s.progressSubscribers[channel] != nil {
		if progressChan, ok := s.progressSubscribers[channel][userID]; ok {
			close(progressChan)
			delete(s.progressSubscribers[channel], userID)
		}
		if len(s.progressSubscribers[channel]) == 0 {
			delete(s.progressSubscribers, channel)
		}
	}

	log.Printf("User %s unsubscribed from channel %s", userID, channel)

	return nil
}

// Broadcast sends a message to all subscribers of a channel
func (s *NotificationService) Broadcast(ctx context.Context, channel string, message *pb.JudgeProgress) error {
	// Publish to Redis for cross-instance communication
	if err := s.store.Publish(ctx, channel, message); err != nil {
		log.Printf("Failed to publish to Redis: %v", err)
	}

	// Also broadcast to local subscribers
	s.progressMu.RLock()
	defer s.progressMu.RUnlock()

	if users, ok := s.progressSubscribers[channel]; ok {
		for userID, progressChan := range users {
			select {
			case progressChan <- message:
			default:
				log.Printf("Channel full for user %s, skipping message", userID)
			}
		}
	}

	return nil
}

// PublishProgress publishes judge progress to Redis pub/sub
// This is called by the judging system to broadcast progress updates
func (s *NotificationService) PublishProgress(ctx context.Context, progress *pb.JudgeProgress) error {
	// Determine the channel based on submission ID
	channel := fmt.Sprintf("judging:result:%s", progress.SubmissionId)

	// Broadcast to all subscribers
	return s.Broadcast(ctx, channel, progress)
}

// PublishTestCaseRun publishes individual test case results
func (s *NotificationService) PublishTestCaseRun(ctx context.Context, submissionID string, run *TestCaseRun) error {
	channel := fmt.Sprintf("judging:run:%s", submissionID)

	data, err := json.Marshal(run)
	if err != nil {
		return err
	}

	return s.redis.Publish(ctx, channel, string(data)).Err()
}

// TestCaseRun represents a single test case run result
type TestCaseRun struct {
	SubmissionID string  `json:"submission_id"`
	TestCaseID   string  `json:"test_case_id"`
	Rank         int     `json:"rank"`
	Verdict      string  `json:"verdict"`
	Runtime      float64 `json:"runtime"`
	Memory       int64   `json:"memory"`
	OutputDiff   string  `json:"output_diff,omitempty"`
	Error        string  `json:"error,omitempty"`
}

// listenToRedisPubSub listens to Redis pub/sub channels for cross-instance communication
func (s *NotificationService) listenToRedisPubSub() {
	ctx := context.Background()

	// Subscribe to judging channels
	pubsub := s.redis.PSubscribe(ctx, "judging:result:*", "judging:run:*")
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("Failed to close pubsub: %v", err)
		}
	}()

	ch := pubsub.Channel()
	for msg := range ch {
		var progress pb.JudgeProgress
		if err := json.Unmarshal([]byte(msg.Payload), &progress); err != nil {
			// Try as TestCaseRun
			var run TestCaseRun
			if err := json.Unmarshal([]byte(msg.Payload), &run); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}
			// Broadcast run to local subscribers
			channel := msg.Channel
			s.progressMu.RLock()
			if users, ok := s.progressSubscribers[channel]; ok {
				for _, progressChan := range users {
					// Convert run to progress format
					runProgress := &pb.JudgeProgress{
						SubmissionId: run.SubmissionID,
						Status:       "running",
						CurrentCase:  int32(run.Rank),
						Verdict:      run.Verdict,
						Runtime:      run.Runtime,
						Memory:       int32(run.Memory),
					}
					select {
					case progressChan <- runProgress:
					default:
					}
				}
			}
			s.progressMu.RUnlock()
			continue
		}

		// Broadcast progress to local subscribers
		channel := msg.Channel
		s.progressMu.RLock()
		if users, ok := s.progressSubscribers[channel]; ok {
			for _, progressChan := range users {
				select {
				case progressChan <- &progress:
				default:
				}
			}
		}
		s.progressMu.RUnlock()
	}
}

// listenToNotificationChannels listens to user notification channels for SSE streaming
func (s *NotificationService) listenToNotificationChannels() {
	ctx := context.Background()

	// Subscribe to all user notification channels
	pubsub := s.redis.PSubscribe(ctx, "notifications:user:*")
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("Failed to close pubsub: %v", err)
		}
	}()

	ch := pubsub.Channel()
	for msg := range ch {
		var notification pb.Notification
		if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
			log.Printf("Failed to unmarshal notification: %v", err)
			continue
		}

		// Extract user ID from channel name
		// Channel format: notifications:user:{userID}
		userID := extractUserIDFromChannel(msg.Channel)

		// Send to stream subscribers
		s.streamMu.RLock()
		if notifChan, ok := s.streamSubscribers[userID]; ok {
			select {
			case notifChan <- &notification:
			default:
				log.Printf("Notification channel full for user %s", userID)
			}
		}
		s.streamMu.RUnlock()
	}
}

// extractUserIDFromChannel extracts user ID from channel name
func extractUserIDFromChannel(channel string) string {
	// Channel format: notifications:user:{userID}
	parts := strings.Split(channel, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// CreateNotification creates a new notification for a user
func (s *NotificationService) CreateNotification(ctx context.Context, notification *pb.Notification) error {
	if notification.Id == "" {
		notification.Id = fmt.Sprintf("notif-%d", time.Now().UnixNano())
	}

	// Save to store
	if err := s.store.Create(ctx, notification); err != nil {
		return err
	}

	// Publish to user's notification channel for real-time delivery
	return s.store.PublishNotification(ctx, notification.UserId, notification)
}

// NotifySubmissionJudged sends a notification when a submission is judged
func (s *NotificationService) NotifySubmissionJudged(ctx context.Context, userID string, submissionID string, verdict string) error {
	notification := &pb.Notification{
		UserId: userID,
		Type:   pb.NotificationType_NOTIFICATION_TYPE_SUBMISSION_JUDGED,
		Title:  "Submission Judged",
		Body:   fmt.Sprintf("Your submission %s has been judged. Result: %s", submissionID, verdict),
		Data: map[string]string{
			"submission_id": submissionID,
			"verdict":       verdict,
		},
	}

	return s.CreateNotification(ctx, notification)
}

// NotifyContestAnnouncement sends a contest announcement to all participants
func (s *NotificationService) NotifyContestAnnouncement(ctx context.Context, contestID string, title string, message string, recipients []string) error {
	for _, userID := range recipients {
		notification := &pb.Notification{
			UserId: userID,
			Type:   pb.NotificationType_NOTIFICATION_TYPE_CONTEST_ANNOUNCEMENT,
			Title:  title,
			Body:   message,
			Data: map[string]string{
				"contest_id": contestID,
			},
		}

		if err := s.CreateNotification(ctx, notification); err != nil {
			log.Printf("Failed to send contest announcement to user %s: %v", userID, err)
		}
	}

	return nil
}

// NotifyContestStarting sends a notification when a contest is about to start
func (s *NotificationService) NotifyContestStarting(ctx context.Context, contestID string, contestName string, minutesBefore int32, recipients []string) error {
	for _, userID := range recipients {
		notification := &pb.Notification{
			UserId: userID,
			Type:   pb.NotificationType_NOTIFICATION_TYPE_CONTEST_STARTING,
			Title:  "Contest Starting Soon",
			Body:   fmt.Sprintf("%s will start in %d minutes", contestName, minutesBefore),
			Data: map[string]string{
				"contest_id":     contestID,
				"contest_name":   contestName,
				"minutes_before": fmt.Sprintf("%d", minutesBefore),
			},
		}

		if err := s.CreateNotification(ctx, notification); err != nil {
			log.Printf("Failed to send contest starting notification to user %s: %v", userID, err)
		}
	}

	return nil
}

// NotifySystemAlert sends a system alert notification to specified users (or all users if empty)
func (s *NotificationService) NotifySystemAlert(ctx context.Context, title string, message string, severity string, recipients []string) error {
	// If no recipients specified, this should broadcast to all users
	// For now, we require explicit recipients
	for _, userID := range recipients {
		notification := &pb.Notification{
			UserId: userID,
			Type:   pb.NotificationType_NOTIFICATION_TYPE_SYSTEM_ALERT,
			Title:  title,
			Body:   message,
			Data: map[string]string{
				"severity": severity,
			},
		}

		if err := s.CreateNotification(ctx, notification); err != nil {
			log.Printf("Failed to send system alert to user %s: %v", userID, err)
		}
	}

	return nil
}

// NotifyClarification sends a clarification notification
func (s *NotificationService) NotifyClarification(ctx context.Context, userID string, contestID string, problemID string, clarification string) error {
	notification := &pb.Notification{
		UserId: userID,
		Type:   pb.NotificationType_NOTIFICATION_TYPE_CLARIFICATION,
		Title:  "Clarification Update",
		Body:   clarification,
		Data: map[string]string{
			"contest_id": contestID,
			"problem_id": problemID,
		},
	}

	return s.CreateNotification(ctx, notification)
}
