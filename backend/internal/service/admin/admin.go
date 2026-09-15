// Package admin backs the operations console (docs/API.md §12).
package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	gmrouter "yingji/backend/internal/engine/genmodel"
	"yingji/backend/internal/pkg/apperr"
	"yingji/backend/internal/pkg/clock"
	"yingji/backend/internal/pkg/idgen"
	"yingji/backend/internal/pkg/jwtx"
	gm "yingji/backend/internal/provider/genmodel"
)

type Service struct {
	db     *gorm.DB
	cfg    *config.Runtime
	signer *jwtx.Signer
	gen    *gmrouter.Router
}

func New(db *gorm.DB, cfg *config.Runtime, signer *jwtx.Signer, gen *gmrouter.Router) *Service {
	return &Service{db: db, cfg: cfg, signer: signer, gen: gen}
}

func (s *Service) Login(ctx context.Context, username, password string) (string, *domain.AdminUser, error) {
	var u domain.AdminUser
	if err := s.db.WithContext(ctx).First(&u, "username = ? AND status = 1", username).Error; err != nil {
		return "", nil, apperr.Unauthorized().WithMessage("用户名或密码错误")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", nil, apperr.Unauthorized().WithMessage("用户名或密码错误")
	}
	now := clock.Now()
	s.db.WithContext(ctx).Model(&u).Update("last_login_at", now)
	tok, _, err := s.signer.Sign(u.ID, u.Role)
	return tok, &u, err
}

func (s *Service) Parse(token string) (id, role string, err error) {
	c, err := s.signer.Parse(token)
	if err != nil {
		return "", "", apperr.Unauthorized()
	}
	return c.Subject, c.Role, nil
}

// Resources maps API resource names to models for generic CRUD.
var Resources = map[string]func() any{
	"categories":  func() any { return &domain.Category{} },
	"specs":       func() any { return &domain.Spec{} },
	"templates":   func() any { return &domain.Template{} },
	"collections": func() any { return &domain.Collection{} },
	"banners":     func() any { return &domain.Banner{} },
}

func sliceFor(resource string) any {
	switch resource {
	case "categories":
		return &[]domain.Category{}
	case "specs":
		return &[]domain.Spec{}
	case "templates":
		return &[]domain.Template{}
	case "collections":
		return &[]domain.Collection{}
	case "banners":
		return &[]domain.Banner{}
	}
	return nil
}

func (s *Service) List(ctx context.Context, resource string, filters map[string]string, page, size int) (any, int64, error) {
	out := sliceFor(resource)
	if out == nil {
		return nil, 0, apperr.NotFound("资源")
	}
	q := s.db.WithContext(ctx).Model(Resources[resource]())
	for k, v := range filters {
		if v != "" {
			q = q.Where(fmt.Sprintf("`%s` = ?", k), v)
		}
	}
	var total int64
	q.Count(&total)
	err := q.Order("sort ASC").Offset((page - 1) * size).Limit(size).Find(out).Error
	return out, total, err
}

// Upsert decodes the JSON body into the resource model and saves it (id generated when empty).
func (s *Service) Upsert(ctx context.Context, resource string, body json.RawMessage, adminID, ip string) (any, error) {
	mk, ok := Resources[resource]
	if !ok {
		return nil, apperr.NotFound("资源")
	}
	model := mk()
	if err := json.Unmarshal(body, model); err != nil {
		return nil, apperr.BadRequest("JSON 无效: " + err.Error())
	}
	switch m := model.(type) {
	case *domain.Category:
		if m.ID == "" {
			m.ID = idgen.New()
		}
	case *domain.Spec:
		if m.ID == "" {
			m.ID = idgen.New()
		}
		if m.IsHot {
			var n int64
			s.db.WithContext(ctx).Model(&domain.Spec{}).Where("is_hot = true AND id <> ?", m.ID).Count(&n)
			if n >= 4 {
				return nil, apperr.BadRequest("常用规格最多 4 个")
			}
		}
		if m.DPI == 0 {
			m.DPI = 300
		}
		m.UpdatedAt = clock.Now()
	case *domain.Template:
		if m.ID == "" {
			m.ID = idgen.New()
		}
		if err := s.ValidateTemplate(ctx, m); err != nil {
			return nil, err
		}
		m.UpdatedAt = clock.Now()
	case *domain.Collection:
		if m.ID == "" {
			m.ID = idgen.New()
		}
	case *domain.Banner:
		if m.ID == "" {
			m.ID = idgen.New()
		}
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error; err != nil {
		return nil, err
	}
	s.audit(ctx, adminID, "upsert:"+resource, resource, body, ip)
	return model, nil
}

func (s *Service) Delete(ctx context.Context, resource, id, adminID, ip string) error {
	mk, ok := Resources[resource]
	if !ok {
		return apperr.NotFound("资源")
	}
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(mk()).Error; err != nil {
		return err
	}
	s.audit(ctx, adminID, "delete:"+resource, resource, json.RawMessage(`{"id":"`+id+`"}`), ip)
	return nil
}

var banned = []string{"最", "第一", "官方", "每一个标准", "100%", "3秒生成", "保证通过"}

// ValidateTemplate checks gen_config against provider capabilities and the banned-word list.
func (s *Service) ValidateTemplate(ctx context.Context, t *domain.Template) error {
	var cfg domain.GenConfig
	if err := t.GenConfig.Into(&cfg); err != nil {
		return apperr.BadRequest("gen_config 无效")
	}
	mode := gm.Mode(cfg.Mode)
	if mode == "" {
		mode = gm.ModeImg2Img
	}
	if cfg.Provider != "" && !s.gen.Has(cfg.Provider) {
		return apperr.BadRequest("provider 不存在: " + cfg.Provider)
	}
	if _, err := s.gen.Resolve(cfg.Provider, cfg.FallbackProvider, mode); err != nil {
		return apperr.BadRequest("没有 provider 支持 mode=" + string(mode))
	}
	texts := []string{t.Name}
	if t.Subtitle != nil {
		texts = append(texts, *t.Subtitle)
	}
	for _, txt := range texts {
		for _, b := range banned {
			if contains(txt, b) {
				return apperr.BadRequest("文案包含禁用词: " + b)
			}
		}
	}
	return nil
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func (s *Service) audit(ctx context.Context, adminID, action, target string, after json.RawMessage, ip string) {
	_ = s.db.WithContext(ctx).Create(&domain.AuditLog{AdminID: adminID, Action: action, TargetType: target, After: domain.JSON(after), IP: ip, CreatedAt: clock.Now()}).Error
}

func (s *Service) SetConfig(ctx context.Context, key string, value json.RawMessage, adminID, ip string) error {
	if _, ok := config.Defaults[key]; !ok {
		return apperr.BadRequest("未知配置项")
	}
	var v any
	if err := json.Unmarshal(value, &v); err != nil {
		return apperr.BadRequest("JSON 无效")
	}
	if err := s.cfg.Set(ctx, key, v, adminID); err != nil {
		return err
	}
	s.audit(ctx, adminID, "config:"+key, "config", value, ip)
	return nil
}

type UserView struct {
	User       domain.User           `json:"user"`
	Identities []domain.UserIdentity `json:"identities"`
	Account    *domain.CreditAccount `json:"account"`
	Ledger     []domain.CreditLedger `json:"ledger"`
}

func (s *Service) FindUser(ctx context.Context, id, openid string) (*UserView, error) {
	var u domain.User
	if id == "" && openid != "" {
		var ident domain.UserIdentity
		if err := s.db.WithContext(ctx).First(&ident, "provider_uid = ?", openid).Error; err != nil {
			return nil, apperr.NotFound("用户")
		}
		id = ident.UserID
	}
	if err := s.db.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil {
		return nil, apperr.NotFound("用户")
	}
	v := &UserView{User: u}
	s.db.WithContext(ctx).Where("user_id = ?", id).Find(&v.Identities)
	var acct domain.CreditAccount
	if s.db.WithContext(ctx).First(&acct, "user_id = ?", id).Error == nil {
		v.Account = &acct
	}
	s.db.WithContext(ctx).Where("user_id = ?", id).Order("created_at DESC").Limit(50).Find(&v.Ledger)
	return v, nil
}

func (s *Service) AuditLogs(ctx context.Context, page, size int) ([]domain.AuditLog, int64, error) {
	q := s.db.WithContext(ctx).Model(&domain.AuditLog{})
	var total int64
	q.Count(&total)
	var rows []domain.AuditLog
	err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

// --- stats ---

type FunnelRow struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

func (s *Service) Funnel(ctx context.Context, from, to time.Time, module string) ([]FunnelRow, error) {
	names := []string{"tab_view", "template_click", "upload_start", "upload_success", "photo_rejected", "generate_click", "no_credits", "ad_show", "ad_ended", "task_created", "task_success", "task_failed", "result_view", "work_saved", "share_click", "share_open", "share_attributed"}
	var rows []FunnelRow
	for _, n := range names {
		q := s.db.WithContext(ctx).Model(&domain.Event{}).Where("name = ? AND created_at >= ? AND created_at < ?", n, from, to)
		if module != "" {
			q = q.Where("props->>'module' = ?", module)
		}
		var c int64
		q.Count(&c)
		rows = append(rows, FunnelRow{Name: n, Count: c})
	}
	return rows, nil
}

type TemplateStat struct {
	TemplateID  string  `json:"template_id"`
	Name        string  `json:"name"`
	Module      string  `json:"module"`
	Tasks       int64   `json:"tasks"`
	Success     int64   `json:"success"`
	Failed      int64   `json:"failed"`
	CostCents   int64   `json:"cost_cents"`
	SuccessRate float64 `json:"success_rate"`
}

func (s *Service) TemplateStats(ctx context.Context, from, to time.Time) ([]TemplateStat, error) {
	var rows []TemplateStat
	err := s.db.WithContext(ctx).Raw(`SELECT t.template_id, tp.name, t.module,
		COUNT(*) AS tasks, SUM(CASE WHEN t.status='success' THEN 1 ELSE 0 END) AS success, SUM(CASE WHEN t.status='failed' THEN 1 ELSE 0 END) AS failed, COALESCE(SUM(t.cost_cents),0) AS cost_cents
		FROM tasks t LEFT JOIN templates tp ON tp.id = t.template_id
		WHERE t.template_id IS NOT NULL AND t.created_at >= ? AND t.created_at < ?
		GROUP BY t.template_id, tp.name, t.module ORDER BY tasks DESC`, from, to).Scan(&rows).Error
	for i := range rows {
		if rows[i].Tasks > 0 {
			rows[i].SuccessRate = float64(rows[i].Success) / float64(rows[i].Tasks)
		}
	}
	return rows, err
}

type ShareStat struct {
	Type     string `json:"type"`
	Module   string `json:"module"`
	Shares   int64  `json:"shares"`
	Opens    int64  `json:"opens"`
	Acquired int64  `json:"acquired_users"`
}

func (s *Service) ShareStats(ctx context.Context, from, to time.Time, module string) ([]ShareStat, error) {
	var rows []ShareStat
	q := s.db.WithContext(ctx).Raw(`SELECT sh.type, sh.module, COUNT(DISTINCT sh.id) AS shares, COALESCE(SUM(sh.opens),0) AS opens,
		(SELECT COUNT(*) FROM users u WHERE u.acquired_share_id IN (SELECT id FROM shares s2 WHERE s2.type = sh.type AND s2.module = sh.module AND s2.created_at >= ? AND s2.created_at < ?)) AS acquired_users
		FROM shares sh WHERE sh.created_at >= ? AND sh.created_at < ? GROUP BY sh.type, sh.module`, from, to, from, to)
	err := q.Scan(&rows).Error
	if module != "" {
		var f []ShareStat
		for _, r := range rows {
			if r.Module == module {
				f = append(f, r)
			}
		}
		rows = f
	}
	return rows, err
}
