// Package photo handles uploads, the synchronous photo check and retention (docs/GENERATION_PIPELINE.md §7.4).
package photo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"time"

	"gorm.io/gorm"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/engine/local"
	"yingji/backend/internal/engine/vision"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/provider/storage"
)

type Service struct {
	db     *gorm.DB
	cfg    *config.Runtime
	store  storage.ObjectStore
	vision *vision.Engine
}

func New(db *gorm.DB, cfg *config.Runtime, store storage.ObjectStore, v *vision.Engine) *Service {
	return &Service{db: db, cfg: cfg, store: store, vision: v}
}

// Upload validates, checks and stores an original. Rejections return PHOTO_REJECTED with reasons.
func (s *Service) Upload(ctx context.Context, userID string, module domain.Module, data []byte) (*domain.Photo, *domain.PhotoCheckResult, error) {
	if int64(len(data)) > int64(s.cfg.Int(ctx, "upload_max_bytes")) {
		return nil, nil, apperr.PayloadTooLarge()
	}
	ct := http.DetectContentType(data)
	if ct != "image/jpeg" && ct != "image/png" && ct != "image/webp" {
		return nil, nil, apperr.BadRequest("仅支持 JPG、PNG 图片")
	}
	img, _, err := local.Decode(data)
	if err != nil {
		return nil, nil, apperr.BadRequest("图片无法读取")
	}
	check := domain.PhotoCheckResult{}
	b := img.Bounds()
	minSide := s.cfg.Int(ctx, "photo_min_side_px")
	if b.Dx() < minSide || b.Dy() < minSide {
		check.Reasons = append(check.Reasons, "low_resolution")
	}
	f, err := s.vision.Detect(ctx, img, data)
	if err != nil {
		return nil, nil, apperr.Transient("VISION_ERROR", err).WithMessage("照片检测暂时不可用，请稍后再试")
	}
	check.Faces = f.Faces
	check.Gender = f.Gender
	switch {
	case f.Faces == 0:
		check.Reasons = append(check.Reasons, "no_face")
	case f.Faces > 1:
		check.Reasons = append(check.Reasons, "multiple_faces")
	default:
		check.FaceBox = [4]int{f.Box.Min.X, f.Box.Min.Y, f.Box.Max.X, f.Box.Max.Y}
		if float64(f.Box.Dy())/float64(b.Dy()) < s.cfg.Float(ctx, "face_min_ratio") {
			check.Reasons = append(check.Reasons, "face_too_small")
		}
		if f.Occluded {
			check.Reasons = append(check.Reasons, "occluded")
		}
		check.Luma = local.MeanLuma(img, f.Box)
		if check.Luma < s.cfg.Float(ctx, "dark_min_luma") {
			check.Reasons = append(check.Reasons, "too_dark")
		}
	}
	check.Blur = local.BlurScore(img)
	if check.Blur < s.cfg.Float(ctx, "blur_min_var") {
		check.Reasons = append(check.Reasons, "blurry")
	}
	check.Passed = len(check.Reasons) == 0
	if !check.Passed {
		return nil, &check, apperr.PhotoRejected(check.Reasons)
	}

	// re-encode: strips EXIF/GPS, normalises orientation
	clean, err := local.EncodeJPEG(img, 95)
	if err != nil {
		return nil, nil, apperr.Internal(err)
	}
	sum := sha256.Sum256(clean)
	now := clock.Now()
	p := &domain.Photo{
		ID: idgen.New(), UserID: userID, Width: b.Dx(), Height: b.Dy(), Bytes: len(clean), SHA256: hex.EncodeToString(sum[:]),
		CheckStatus: domain.CheckPassed, CheckResult: domain.MustJSON(check), ModerationStatus: domain.ModPending,
		ExpiresAt: now.Add(time.Duration(s.cfg.Int(ctx, "photo_retention_days")) * 24 * time.Hour), CreatedAt: now,
	}
	p.ObjectKey = "originals/" + userID + "/" + p.ID + ".jpg"
	if err := s.store.Put(ctx, p.ObjectKey, bytes.NewReader(clean), int64(len(clean)), "image/jpeg"); err != nil {
		return nil, nil, apperr.Internal(err)
	}
	if err := s.db.WithContext(ctx).Create(p).Error; err != nil {
		return nil, nil, err
	}
	return p, &check, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (*domain.Photo, error) {
	var p domain.Photo
	if err := s.db.WithContext(ctx).First(&p, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, apperr.NotFound("照片")
	}
	return &p, nil
}

// Load returns the photo by id regardless of owner (worker use).
func (s *Service) Load(ctx context.Context, id string) (*domain.Photo, error) {
	var p domain.Photo
	if err := s.db.WithContext(ctx).Unscoped().First(&p, "id = ?", id).Error; err != nil {
		return nil, apperr.NotFound("照片")
	}
	return &p, nil
}

func (s *Service) Bytes(ctx context.Context, p *domain.Photo) ([]byte, error) {
	rc, err := s.store.Get(ctx, p.ObjectKey)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (s *Service) URL(ctx context.Context, p *domain.Photo) string {
	return storage.URLFor(ctx, s.store, p.ObjectKey, 30*time.Minute)
}

func (s *Service) List(ctx context.Context, userID string, page, size int) ([]domain.Photo, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	q := s.db.WithContext(ctx).Model(&domain.Photo{}).Where("user_id = ?", userID)
	var total int64
	q.Count(&total)
	var rows []domain.Photo
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	_ = s.store.Delete(ctx, p.ObjectKey)
	return s.db.WithContext(ctx).Delete(p).Error
}

// Cleanup removes originals past their expiry (docs/DECISIONS.md D-16).
func (s *Service) Cleanup(ctx context.Context) (int, error) {
	var rows []domain.Photo
	if err := s.db.WithContext(ctx).Where("expires_at < ?", clock.Now()).Limit(500).Find(&rows).Error; err != nil {
		return 0, err
	}
	n := 0
	for _, p := range rows {
		if err := s.store.Delete(ctx, p.ObjectKey); err != nil && !errors.Is(err, context.Canceled) {
			continue
		}
		if err := s.db.WithContext(ctx).Delete(&p).Error; err == nil {
			n++
		}
	}
	return n, nil
}

func (s *Service) SetModeration(ctx context.Context, id string, status domain.ModerationStatus) error {
	return s.db.WithContext(ctx).Model(&domain.Photo{}).Where("id = ?", id).Update("moderation_status", status).Error
}
