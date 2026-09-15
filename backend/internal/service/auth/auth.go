// Package auth handles login via identity providers and JWT issuance (docs/DECISIONS.md D-23).
package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/pkg/jwtx"
	"yingji/backend/internal/provider/wechat"
)

type Service struct {
	db     *gorm.DB
	wx     *wechat.Client
	signer *jwtx.Signer
	dev    bool
}

func New(db *gorm.DB, wx *wechat.Client, signer *jwtx.Signer, dev bool) *Service {
	return &Service{db: db, wx: wx, signer: signer, dev: dev}
}

type LoginResult struct {
	Token     string
	ExpiresIn int
	User      *domain.User
	IsNew     bool
}

func (s *Service) Login(ctx context.Context, provider, code, shareID string) (*LoginResult, error) {
	var uid, unionID string
	switch provider {
	case "wechat_mp", "wechat_app":
		sess, err := s.wx.Code2Session(ctx, code)
		if err != nil {
			return nil, apperr.Unauthorized().WithMessage("微信登录失败").WithCause(err)
		}
		uid, unionID = sess.OpenID, sess.UnionID
	case "h5_dev":
		if !s.dev {
			return nil, apperr.Forbidden()
		}
		uid = "dev-" + code
	default:
		return nil, apperr.BadRequest("unsupported provider")
	}

	var user domain.User
	isNew := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ident domain.UserIdentity
		err := tx.First(&ident, "provider = ? AND provider_uid = ?", provider, uid).Error
		switch {
		case err == nil:
			return tx.First(&user, "id = ?", ident.UserID).Error
		case errors.Is(err, gorm.ErrRecordNotFound):
			// link by unionid across providers
			if unionID != "" {
				var other domain.UserIdentity
				if tx.First(&other, "union_id = ?", unionID).Error == nil {
					if err := tx.First(&user, "id = ?", other.UserID).Error; err != nil {
						return err
					}
				}
			}
			now := clock.Now()
			if user.ID == "" {
				user = domain.User{ID: idgen.New(), Status: 1, LastLoginAt: now, CreatedAt: now, UpdatedAt: now}
				if shareID != "" {
					user.AcquiredShareID = &shareID
				}
				if err := tx.Create(&user).Error; err != nil {
					return err
				}
				isNew = true
			}
			ident = domain.UserIdentity{ID: idgen.New(), UserID: user.ID, Provider: provider, ProviderUID: uid, CreatedAt: now}
			if unionID != "" {
				ident.UnionID = &unionID
			}
			return tx.Create(&ident).Error
		default:
			return err
		}
	})
	if err != nil {
		return nil, err
	}
	if user.Status != 1 {
		return nil, apperr.Forbidden().WithMessage("账号不可用")
	}
	s.db.WithContext(ctx).Model(&user).Update("last_login_at", clock.Now())
	if isNew && shareID != "" {
		s.attribute(ctx, &user, shareID)
	}
	tok, exp, err := s.signer.Sign(user.ID, "")
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &LoginResult{Token: tok, ExpiresIn: int(time.Until(exp).Seconds()), User: &user, IsNew: isNew}, nil
}

func (s *Service) attribute(ctx context.Context, u *domain.User, shareID string) {
	var sh domain.Share
	if s.db.WithContext(ctx).First(&sh, "id = ? AND status = 'active'", shareID).Error != nil {
		return
	}
	if sh.UserID == u.ID {
		return
	}
	s.db.WithContext(ctx).Model(u).Update("acquired_share_id", shareID)
	props := domain.MustJSON(map[string]any{"share_id": shareID, "sharer_id": sh.UserID})
	s.db.WithContext(ctx).Create(&domain.Event{UserID: &u.ID, Name: "share_attributed", Props: props, CreatedAt: clock.Now()})
}

// Parse validates a user token and returns the user id.
func (s *Service) Parse(token string) (string, error) {
	c, err := s.signer.Parse(token)
	if err != nil {
		return "", apperr.Unauthorized()
	}
	return c.Subject, nil
}

// OpenID returns the WeChat Mini Program openid of a user, if any.
func (s *Service) OpenID(ctx context.Context, userID string) (string, error) {
	var ident domain.UserIdentity
	if err := s.db.WithContext(ctx).First(&ident, "user_id = ? AND provider = ?", userID, "wechat_mp").Error; err != nil {
		return "", err
	}
	return ident.ProviderUID, nil
}

// UserByOpenID resolves the openid of a WeChat identity to a user id.
func (s *Service) UserByOpenID(ctx context.Context, openid string) (string, error) {
	var ident domain.UserIdentity
	if err := s.db.WithContext(ctx).First(&ident, "provider_uid = ? AND provider IN ('wechat_mp','wechat_app')", openid).Error; err != nil {
		return "", err
	}
	return ident.UserID, nil
}
