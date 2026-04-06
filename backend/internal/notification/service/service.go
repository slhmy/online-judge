package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	pb "github.com/online-judge/backend/gen/go/notification/v1"
	"github.com/online-judge/backend/internal/notification/store"
)

// NotificationService implements the notification gRPC service
type NotificationService struct {
	pb.UnimplementedNotificationServiceServer
	redis *redis.Client
	store *store.NotificationStore

	// Local subscriber management for WebSocket connections
	// channel -> userID -> progressChan
	subscribers map[string]map[string]chan *pb.JudgeProgress
	mu          sync.RWMutex
}

// NewNotificationService creates a new notification service
func NewNotificationService(redis *redis.Client) *NotificationService {
	s := &NotificationService{
		redis:      redis,
		store:      store.NewNotificationStore(redis),
		subscribers: make(map[string]map[string]chan *pb.JudgeProgress),
	}

	// Start Redis pub/sub listener for cross-instance communication
	go s.listenToRedisPubSub()

	return s
}

// GetNotifications retrieves notifications for a user
func (s *NotificationService) GetNotifications(ctx context.Context, req *pb.GetNotificationsRequest) (*pb.GetNotificationsResponse, error) {
	// Get user ID from context (extracted from JWT token in interceptor)
	userID := ctx.Value("user_id")
	if userID == nil {
		return nil, fmt.Errorf("user not authenticated")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}

	notifications, err := s.store.List(ctx, userID.(string), limit, req.UnreadOnly)
	if err != nil {
		return nil, err
	}

	return &pb.GetNotificationsResponse{
		Notifications: notifications,
	}, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
	userID := ctx.Value("user_id")
	if userID == nil {
		return nil, fmt.Errorf("user not authenticated")
	}

	err := s.store.MarkAsRead(ctx, userID.(string), req.Id)
	if err != nil {
		return &pb.MarkAsReadResponse{Success: false}, err
	}

	return &pb.MarkAsReadResponse{Success: true}, nil
}

// Subscribe subscribes a user to a notification channel
// Returns a channel that will receive progress updates
func (s *NotificationService) Subscribe(ctx context.Context, userID string, channel string) (chan *pb.JudgeProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to Redis subscriber list for cross-instance communication
	if err := s.store.AddSubscriber(ctx, channel, userID); err != nil {
		log.Printf("Failed to add subscriber to Redis: %v", err)
	}

	// Create local channel for this user
	if s.subscribers[channel] == nil {
		s.subscribers[channel] = make(map[string]chan *pb.JudgeProgress)
	}

	progressChan := make(chan *pb.JudgeProgress, 100)
	s.subscribers[channel][userID] = progressChan

	log.Printf("User %s subscribed to channel %s", userID, channel)

	return progressChan, nil
}

// Unsubscribe removes a user from a notification channel
func (s *NotificationService) Unsubscribe(ctx context.Context, userID string, channel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from Redis subscriber list
	if err := s.store.RemoveSubscriber(ctx, channel, userID); err != nil {
		log.Printf("Failed to remove subscriber from Redis: %v", err)
	}

	// Close and remove local channel
	if s.subscribers[channel] != nil {
		if progressChan, ok := s.subscribers[channel][userID]; ok {
			close(progressChan)
			delete(s.subscribers[channel], userID)
		}
		if len(s.subscribers[channel]) == 0 {
			delete(s.subscribers, channel)
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	if users, ok := s.subscribers[channel]; ok {
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
	defer pubsub.Close()

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
			s.mu.RLock()
			if users, ok := s.subscribers[channel]; ok {
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
			s.mu.RUnlock()
			continue
		}

		// Broadcast progress to local subscribers
		channel := msg.Channel
		s.mu.RLock()
		if users, ok := s.subscribers[channel]; ok {
			for _, progressChan := range users {
				select {
				case progressChan <- &progress:
				default:
				}
			}
		}
		s.mu.RUnlock()
	}
}

// CreateNotification creates a new notification for a user
func (s *NotificationService) CreateNotification(ctx context.Context, notification *pb.Notification) error {
	if notification.Id == "" {
		notification.Id = fmt.Sprintf("notif-%d", time.Now().UnixNano())
	}

	return s.store.Create(ctx, notification)
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