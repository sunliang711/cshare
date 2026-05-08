package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP2PService_MessageExchange(t *testing.T) {
	svc := NewP2PService(zerolog.Nop())
	ctx := context.Background()

	session, err := svc.CreateSession(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, session.SessionID)

	payload := json.RawMessage(`{"type":"offer","sdp":"test"}`)
	msg, err := svc.PostMessage(ctx, session.SessionID, P2PMessageRequest{
		From:    "sender",
		To:      "receiver",
		Type:    "offer",
		Payload: payload,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), msg.Seq)

	messages, err := svc.WaitMessages(ctx, session.SessionID, "receiver", 0, 0)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "sender", messages[0].From)
	assert.JSONEq(t, string(payload), string(messages[0].Payload))

	empty, err := svc.WaitMessages(ctx, session.SessionID, "receiver", messages[0].Seq, 0)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestP2PService_LongPollWakesOnMessage(t *testing.T) {
	svc := NewP2PService(zerolog.Nop())
	ctx := context.Background()

	session, err := svc.CreateSession(ctx)
	require.NoError(t, err)

	resultCh := make(chan []string, 1)
	errCh := make(chan error, 1)
	go func() {
		messages, err := svc.WaitMessages(ctx, session.SessionID, "sender", 0, time.Second)
		if err != nil {
			errCh <- err
			return
		}
		types := make([]string, 0, len(messages))
		for _, msg := range messages {
			types = append(types, msg.Type)
		}
		resultCh <- types
	}()

	_, err = svc.PostMessage(ctx, session.SessionID, P2PMessageRequest{
		From:    "receiver",
		To:      "sender",
		Type:    "answer",
		Payload: json.RawMessage(`{"type":"answer"}`),
	})
	require.NoError(t, err)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case types := <-resultCh:
		assert.Equal(t, []string{"answer"}, types)
	case <-time.After(time.Second):
		t.Fatal("long poll did not wake")
	}
}

func TestP2PService_CloseSession(t *testing.T) {
	svc := NewP2PService(zerolog.Nop())
	ctx := context.Background()

	session, err := svc.CreateSession(ctx)
	require.NoError(t, err)

	closed, err := svc.CloseSession(ctx, session.SessionID)
	require.NoError(t, err)
	assert.True(t, closed)

	_, err = svc.WaitMessages(ctx, session.SessionID, "sender", 0, 0)
	require.Error(t, err)
}
