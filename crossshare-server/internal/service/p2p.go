package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/rs/zerolog"

	apperr "crossshare-server/internal/errors"
	"crossshare-server/internal/keygen"
	"crossshare-server/internal/model"
)

const (
	p2pSessionIDLength = 16
	p2pSessionTTL      = 5 * time.Minute
	p2pMaxMessages     = 256
	p2pMaxPayloadBytes = 64 << 10
	p2pMaxWait         = 25 * time.Second
)

type P2PService struct {
	mu       sync.Mutex
	sessions map[string]*p2pSession
	logger   zerolog.Logger
}

type p2pSession struct {
	id        string
	createdAt int64
	expireAt  int64
	nextSeq   int64
	messages  []model.P2PMessage
	updated   chan struct{}
}

type P2PMessageRequest struct {
	From    string
	To      string
	Type    string
	Payload json.RawMessage
}

func NewP2PService(logger zerolog.Logger) *P2PService {
	return &P2PService{
		sessions: make(map[string]*p2pSession),
		logger:   logger.With().Str("component", "p2p").Logger(),
	}
}

func (s *P2PService) CreateSession(ctx context.Context) (*model.P2PSessionResult, error) {
	now := time.Now().Unix()
	expireAt := now + int64(p2pSessionTTL/time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(now)

	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		id, err := keygen.Generate(p2pSessionIDLength)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to generate p2p session id")
			return nil, apperr.ErrStorageInternal
		}
		if _, exists := s.sessions[id]; exists {
			continue
		}

		s.sessions[id] = &p2pSession{
			id:        id,
			createdAt: now,
			expireAt:  expireAt,
			nextSeq:   1,
			updated:   make(chan struct{}),
		}
		return &model.P2PSessionResult{
			SessionID: id,
			TTL:       int(p2pSessionTTL / time.Second),
			ExpireAt:  expireAt,
		}, nil
	}

	return nil, apperr.ErrStorageInternal
}

func (s *P2PService) PostMessage(ctx context.Context, sessionID string, req P2PMessageRequest) (*model.P2PMessage, error) {
	if len(req.Payload) > p2pMaxPayloadBytes {
		return nil, apperr.ErrPayloadTooLarge
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(time.Now().Unix())

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	session := s.sessions[sessionID]
	if session == nil {
		return nil, apperr.ErrNotFound
	}

	msg := model.P2PMessage{
		Seq:     session.nextSeq,
		From:    req.From,
		To:      req.To,
		Type:    req.Type,
		Payload: append(json.RawMessage(nil), req.Payload...),
	}
	session.nextSeq++
	session.messages = append(session.messages, msg)
	if len(session.messages) > p2pMaxMessages {
		session.messages = session.messages[len(session.messages)-p2pMaxMessages:]
	}
	s.notifyLocked(session)

	return &msg, nil
}

func (s *P2PService) WaitMessages(ctx context.Context, sessionID string, to string, after int64, wait time.Duration) ([]model.P2PMessage, error) {
	if wait < 0 || wait > p2pMaxWait {
		wait = p2pMaxWait
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	waiting := false
	for {
		s.mu.Lock()
		s.cleanupExpiredLocked(time.Now().Unix())
		session := s.sessions[sessionID]
		if session == nil {
			s.mu.Unlock()
			if waiting {
				return []model.P2PMessage{}, nil
			}
			return nil, apperr.ErrNotFound
		}

		messages := collectP2PMessages(session.messages, to, after)
		if len(messages) > 0 || wait == 0 {
			s.mu.Unlock()
			return messages, nil
		}

		updated := session.updated
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return []model.P2PMessage{}, nil
		case <-updated:
			waiting = true
		}
	}
}

func (s *P2PService) CloseSession(ctx context.Context, sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	session := s.sessions[sessionID]
	if session == nil {
		return false, nil
	}
	delete(s.sessions, sessionID)
	s.notifyLocked(session)
	return true, nil
}

func collectP2PMessages(messages []model.P2PMessage, to string, after int64) []model.P2PMessage {
	result := make([]model.P2PMessage, 0)
	for _, msg := range messages {
		if msg.Seq <= after || msg.To != to {
			continue
		}
		result = append(result, msg)
	}
	return result
}

func (s *P2PService) cleanupExpiredLocked(now int64) {
	for id, session := range s.sessions {
		if session.expireAt > now {
			continue
		}
		delete(s.sessions, id)
		s.notifyLocked(session)
	}
}

func (s *P2PService) notifyLocked(session *p2pSession) {
	close(session.updated)
	session.updated = make(chan struct{})
}
