package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"crossshare-server/internal/crypto"
	"crossshare-server/internal/db"
	"crossshare-server/internal/middleware"
)

const adminSessionTTL = 24 * time.Hour // 与现有管理员登录有效期保持一致。

type AdminError struct {
	Status  int
	Code    string
	Message string
}

func (e *AdminError) Error() string { return e.Message }

func adminError(status int, code, message string) error {
	return &AdminError{Status: status, Code: code, Message: message}
}

type AdminService struct {
	store  *db.Store
	secret []byte
}

func NewAdminService(store *db.Store, cipher *crypto.Cipher) *AdminService {
	return &AdminService{store: store, secret: middleware.AdminJWTSecret(cipher)}
}

func (s *AdminService) prepareSession(admin *db.Admin, ip, userAgent string) (*db.AdminSession, string, error) {
	now := time.Now().UTC().Truncate(time.Second)
	session := &db.AdminSession{ID: crypto.NewULID(), AdminID: admin.ID, ClientIP: ip, UserAgent: userAgent,
		CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(adminSessionTTL)}
	token, err := middleware.SignAdminToken(s.secret, admin, session)
	if err != nil {
		return nil, "", err
	}
	session.TokenHash = db.AdminTokenHash(token)
	return session, token, nil
}

func (s *AdminService) CreateSession(ctx context.Context, admin *db.Admin, ip, userAgent string) (string, error) {
	session, token, err := s.prepareSession(admin, ip, userAgent)
	if err != nil {
		return "", err
	}
	if err := s.store.CreateAdminSession(ctx, admin, session); err != nil {
		return "", err
	}
	return token, nil
}

type AdminSessionView struct {
	ID         string    `json:"id"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Current    bool      `json:"current"`
}

func (s *AdminService) ListSessions(ctx context.Context, adminID, currentID string) ([]AdminSessionView, error) {
	sessions, err := s.store.ListAdminSessions(ctx, adminID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	out := make([]AdminSessionView, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, AdminSessionView{ID: session.ID, ClientIP: session.ClientIP, UserAgent: session.UserAgent,
			CreatedAt: session.CreatedAt, LastSeenAt: session.LastSeenAt, ExpiresAt: session.ExpiresAt, Current: session.ID == currentID})
	}
	return out, nil
}

func (s *AdminService) RevokeSession(ctx context.Context, adminID, id string) error {
	return s.store.DeleteAdminSession(ctx, adminID, id)
}

func validateAdminPassword(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return adminError(http.StatusBadRequest, "weak_password", "password must contain 8 to 72 bytes")
	}
	return nil
}

func (s *AdminService) ChangePassword(ctx context.Context, admin *db.Admin, old, next, ip, userAgent string) (string, error) {
	if (!admin.MustChangePassword || old != "") && !crypto.VerifyPassword(admin.PasswordHash, old) {
		return "", adminError(http.StatusBadRequest, "invalid_password", "current password is incorrect")
	}
	if err := validateAdminPassword(next); err != nil {
		return "", err
	}
	hash, err := crypto.HashPassword(next)
	if err != nil {
		return "", err
	}
	updated := *admin
	updated.PasswordHash, updated.MustChangePassword = hash, false
	session, token, err := s.prepareSession(&updated, ip, userAgent)
	if err != nil {
		return "", err
	}
	if err := s.store.ChangeAdminPasswordWithSession(ctx, admin, hash, session); err != nil {
		return "", err
	}
	*admin = updated
	return token, nil
}

func authorizeAccountChange(actor *db.Admin, currentPassword string) error {
	if actor == nil || actor.Role != db.RoleAdmin {
		return adminError(http.StatusForbidden, "forbidden", "administrator role required")
	}
	if !crypto.VerifyPassword(actor.PasswordHash, currentPassword) {
		return adminError(http.StatusForbidden, "wrong_current_password", "current password verification failed")
	}
	return nil
}

func validateAdminRole(role string) error {
	if role != db.RoleAdmin && role != db.RoleViewer {
		return adminError(http.StatusBadRequest, "invalid_role", "role must be admin or viewer")
	}
	return nil
}

type CreateAdminInput struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	Role            string `json:"role"`
	IsActive        *bool  `json:"is_active"`
	CurrentPassword string `json:"current_password"`
}

func (s *AdminService) CreateAccount(ctx context.Context, actor *db.Admin, in CreateAdminInput) (*db.Admin, error) {
	if err := authorizeAccountChange(actor, in.CurrentPassword); err != nil {
		return nil, err
	}
	in.Username = strings.TrimSpace(in.Username)
	if len(in.Username) < 3 || len(in.Username) > 64 {
		return nil, adminError(http.StatusBadRequest, "invalid_username", "username must contain 3 to 64 bytes")
	}
	if in.Role == "" {
		in.Role = db.RoleViewer
	}
	if err := validateAdminRole(in.Role); err != nil {
		return nil, err
	}
	if err := validateAdminPassword(in.Password); err != nil {
		return nil, err
	}
	hash, err := crypto.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	active := in.IsActive == nil || *in.IsActive
	admin := &db.Admin{ID: crypto.NewULID(), Username: in.Username, PasswordHash: hash, Role: in.Role, IsActive: active}
	if err := s.store.CreateAdminByActor(ctx, actor, admin); err != nil {
		return nil, err
	}
	return admin, nil
}

type UpdateAdminInput struct {
	IsActive        *bool  `json:"is_active"`
	Role            string `json:"role"`
	Password        string `json:"password"`
	CurrentPassword string `json:"current_password"`
}

func (s *AdminService) UpdateAccount(ctx context.Context, actor *db.Admin, id string, in UpdateAdminInput, ip, userAgent string) (*db.Admin, string, error) {
	if err := authorizeAccountChange(actor, in.CurrentPassword); err != nil {
		return nil, "", err
	}
	target, err := s.store.GetAdminByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if in.Role == "" {
		in.Role = target.Role
	}
	if err := validateAdminRole(in.Role); err != nil {
		return nil, "", err
	}
	active := target.IsActive
	if in.IsActive != nil {
		active = *in.IsActive
	}
	if id == actor.ID && !active {
		return nil, "", adminError(http.StatusBadRequest, "cannot_disable_self", "cannot disable the currently logged-in account")
	}
	hash := ""
	if in.Password != "" {
		if err := validateAdminPassword(in.Password); err != nil {
			return nil, "", err
		}
		hash, err = crypto.HashPassword(in.Password)
		if err != nil {
			return nil, "", err
		}
	}
	target.Role, target.IsActive = in.Role, active
	var replacement *db.AdminSession
	token := ""
	if hash != "" {
		target.PasswordHash, target.MustChangePassword = hash, false
		if id == actor.ID && in.Role == actor.Role {
			replacement, token, err = s.prepareSession(target, ip, userAgent)
			if err != nil {
				return nil, "", err
			}
		}
	}
	if err := s.store.UpdateAdminAccount(ctx, actor, id, in.Role, active, hash, replacement); err != nil {
		return nil, "", err
	}
	return target, token, nil
}
