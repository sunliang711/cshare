package audit

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"crossshare-server/internal/db"
	"crossshare-server/internal/settings"
)

var Module = fx.Options(
	fx.Provide(ProvideRecorder),
)

type Event struct {
	ActorType    string // admin | client | system
	ActorID      string
	ActorName    string
	Action       string
	ResourceType string
	ResourceID   string
	ShareKey     string
	Status       string // success | failure
	IPAddress    string
	UserAgent    string
	RequestID    string
	Detail       any
}

type Recorder struct {
	store    *db.Store
	settings *settings.Manager
	logger   zerolog.Logger
	ch       chan db.AuditLog
	done     chan struct{}
	once     sync.Once
}

func ProvideRecorder(lc fx.Lifecycle, store *db.Store, settingsMgr *settings.Manager, logger zerolog.Logger) *Recorder {
	r := NewRecorder(store, settingsMgr, logger)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			r.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			r.Stop()
			return nil
		},
	})
	return r
}

func NewRecorder(store *db.Store, settingsMgr *settings.Manager, logger zerolog.Logger) *Recorder {
	return &Recorder{
		store:    store,
		settings: settingsMgr,
		logger:   logger.With().Str("component", "audit").Logger(),
		ch:       make(chan db.AuditLog, 1024),
		done:     make(chan struct{}),
	}
}

func (r *Recorder) Start() {
	go func() {
		defer close(r.done)
		for entry := range r.ch {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if err := r.store.InsertAuditLog(ctx, &entry); err != nil {
				r.logger.Warn().Err(err).Str("action", entry.Action).Msg("failed to write audit log")
			}
			cancel()
		}
	}()
}

func (r *Recorder) Stop() {
	r.once.Do(func() {
		close(r.ch)
		<-r.done
	})
}

func (r *Recorder) Record(ev Event) {
	if r == nil || r.store == nil {
		return
	}
	if ev.ActorType == "client" && r.settings != nil && !r.settings.Get().AuditClientOps {
		return
	}
	entry := r.toModel(ev)
	select {
	case r.ch <- entry:
	default:
		r.logger.Warn().Str("action", entry.Action).Msg("audit queue full, dropping event")
	}
}

func (r *Recorder) RecordFromGin(c *gin.Context, ev Event) {
	if ev.IPAddress == "" {
		ev.IPAddress = c.ClientIP()
	}
	if ev.UserAgent == "" {
		ev.UserAgent = c.Request.UserAgent()
	}
	if ev.RequestID == "" {
		ev.RequestID = c.GetString("request_id")
	}
	r.Record(ev)
}

func (r *Recorder) RecordSync(ctx context.Context, ev Event) error {
	if r == nil || r.store == nil {
		return nil
	}
	if ev.ActorType == "client" && r.settings != nil && !r.settings.Get().AuditClientOps {
		return nil
	}
	entry := r.toModel(ev)
	return r.store.InsertAuditLog(ctx, &entry)
}

func (r *Recorder) toModel(ev Event) db.AuditLog {
	detailJSON := "{}"
	if ev.Detail != nil {
		if b, err := json.Marshal(ev.Detail); err == nil {
			detailJSON = string(b)
		}
	}
	status := ev.Status
	if status == "" {
		status = "success"
	}
	return db.AuditLog{
		ActorType:    ev.ActorType,
		ActorID:      ev.ActorID,
		ActorName:    ev.ActorName,
		Action:       ev.Action,
		ResourceType: ev.ResourceType,
		ResourceID:   ev.ResourceID,
		ShareKey:     ev.ShareKey,
		IPAddress:    ev.IPAddress,
		UserAgent:    ev.UserAgent,
		RequestID:    ev.RequestID,
		Status:       status,
		DetailJSON:   detailJSON,
		CreatedAt:    time.Now().UTC(),
	}
}
