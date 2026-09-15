// Package seed loads the initial catalogue, configuration and admin user.
// Sizes marked "示例数据，上线前核实" must be verified before production (ui/UI_REVIEW.md UI-16).
package seed

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingji/backend/internal/config"
	"yingji/backend/internal/domain"
	"yingji/backend/internal/provider/storage"
)

const VERIFY = "示例数据，上线前核实"

func str(s string) *string { return &s }

// Run is idempotent: rows are upserted by fixed ids.
func Run(ctx context.Context, db *gorm.DB, store storage.ObjectStore, assetsDir string, adminUser, adminPass string) error {
	if err := uploadAssets(ctx, store, assetsDir); err != nil {
		return err
	}
	now := time.Now().UTC()
	up := func(rows any) error {
		return db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(rows).Error
	}

	specCats := []domain.Category{
		{ID: "sc_common", Module: domain.ModuleIDPhoto, Kind: "spec", Name: "常用证件照", Sort: 1, Status: 1},
		{ID: "sc_exam", Module: domain.ModuleIDPhoto, Kind: "spec", Name: "考试报名", Sort: 2, Status: 1},
		{ID: "sc_job", Module: domain.ModuleIDPhoto, Kind: "spec", Name: "求职招聘", Sort: 3, Status: 1},
		{ID: "sc_cert", Module: domain.ModuleIDPhoto, Kind: "spec", Name: "资格认证", Sort: 4, Status: 1},
		{ID: "sc_visa", Module: domain.ModuleIDPhoto, Kind: "spec", Name: "签证", Sort: 5, Status: 1},
		{ID: "sc_passport", Module: domain.ModuleIDPhoto, Kind: "spec", Name: "护照", Sort: 6, Status: 1},
		{ID: "sc_school", Module: domain.ModuleIDPhoto, Kind: "spec", Name: "学籍", Sort: 7, Status: 1},
	}
	if err := up(&specCats); err != nil {
		return err
	}

	bgAll := domain.MustJSON([]string{"#FFFFFF", "#438EDB", "#FF0000", "#808080"})
	bgWhite := domain.MustJSON([]string{"#FFFFFF"})
	spec := func(id, name string, wmm, hmm float64, wpx, hpx int, cat string, bg string, allowed domain.JSON, hot bool, note string, sort int) domain.Spec {
		s := domain.Spec{ID: id, CategoryID: cat, Name: name, WidthMM: wmm, HeightMM: hmm, WidthPx: wpx, HeightPx: hpx, DPI: 300,
			BgDefault: bg, BgAllowed: allowed, CropRule: domain.JSON("{}"), IsHot: hot, Sort: sort, Status: 1, CreatedAt: now, UpdatedAt: now}
		if note != "" {
			s.Note = str(note)
			if note == VERIFY {
				s.SourceNote = str("unverified")
			}
		}
		return s
	}
	specs := []domain.Spec{
		spec("sp_1inch", "一寸", 25, 35, 295, 413, "sc_common", "#438EDB", bgAll, true, "国内通用", 1),
		spec("sp_2inch", "二寸", 35, 49, 413, 579, "sc_common", "#438EDB", bgAll, true, "国内通用", 2),
		spec("sp_s2inch", "小二寸", 35, 45, 413, 531, "sc_common", "#438EDB", bgAll, true, "国内通用", 3),
		spec("sp_b1inch", "大一寸", 33, 48, 390, 567, "sc_common", "#438EDB", bgAll, true, "港澳通行证等", 4),
		spec("sp_civil", "公务员报名", 25, 35, 295, 413, "sc_exam", "#438EDB", bgAll, false, VERIFY, 10),
		spec("sp_cet", "四六级报名", 25, 35, 295, 413, "sc_exam", "#438EDB", bgAll, false, VERIFY, 11),
		spec("sp_resume", "简历照", 25, 35, 295, 413, "sc_job", "#FFFFFF", bgAll, false, "", 20),
		spec("sp_teacher", "教师资格", 35, 45, 413, 531, "sc_cert", "#438EDB", bgAll, false, VERIFY, 30),
		spec("sp_schengen", "申根签证", 35, 45, 413, 531, "sc_visa", "#FFFFFF", bgWhite, false, "", 40),
		spec("sp_us", "美国签证", 51, 51, 600, 600, "sc_visa", "#FFFFFF", bgWhite, false, "", 41),
		spec("sp_jp", "日本签证", 45, 45, 531, 531, "sc_visa", "#FFFFFF", bgWhite, false, "", 42),
		spec("sp_passport", "中国护照", 33, 48, 390, 567, "sc_passport", "#FFFFFF", bgWhite, false, "", 50),
		spec("sp_student", "学籍照", 35, 45, 413, 531, "sc_school", "#438EDB", bgAll, false, VERIFY, 60),
	}
	if err := up(&specs); err != nil {
		return err
	}

	asset := func(name string) string { return "assets/seed/" + name + ".jpg" }
	cat := func(id string, m domain.Module, name, icon, cover, desc string, sort int) domain.Category {
		c := domain.Category{ID: id, Module: m, Kind: "template", Name: name, Sort: sort, Status: 1}
		if icon != "" {
			c.Icon = str(icon)
		}
		if cover != "" {
			c.CoverKey = str(asset(cover))
		}
		if desc != "" {
			c.Desc = str(desc)
		}
		return c
	}
	cats := []domain.Category{
		cat("c_interview", domain.ModulePro, "求职面试", "briefcase", "", "正装·纯色背景", 1),
		cat("c_business", domain.ModulePro, "商务职场", "users", "", "西装·办公室", 2),
		cat("c_lecturer", domain.ModulePro, "讲师头像", "presentation", "", "半身·浅色背景", 3),
		cat("c_website", domain.ModulePro, "企业官网", "building", "", "正装·灰色背景", 4),
		cat("c_consultant", domain.ModulePro, "顾问", "user-check", "", "正装·浅色背景", 5),
		cat("c_doctor", domain.ModulePro, "医生", "shield", "", "白大褂·白底", 6),
		cat("c_korean", domain.ModulePortrait, "韩系清透", "", "portrait_korean", "自然光·户外", 1),
		cat("c_french", domain.ModulePortrait, "法式氛围", "", "portrait_french", "街拍·胶片感", 2),
		cat("c_chinese", domain.ModulePortrait, "新中式", "", "portrait_chinese2", "国风·庭院", 3),
		cat("c_birthday", domain.ModulePortrait, "生日写真", "", "portrait_birthday2", "气球·蛋糕", 4),
		cat("c_campus", domain.ModulePortrait, "校园", "", "", "校园·日常", 5),
		cat("c_travel", domain.ModulePortrait, "旅行", "", "", "海边·山野", 6),
		cat("c_autumn", domain.ModulePortrait, "秋日", "", "", "暖色·户外", 7),
		cat("c_hk", domain.ModulePortrait, "港风", "", "", "复古·胶片", 8),
		cat("c_wechat", domain.ModuleAvatar, "微信头像", "message-circle", "", "1:1·近景", 1),
		cat("c_premium", domain.ModuleAvatar, "高级感", "crown", "", "1:1·影棚光", 2),
		cat("c_fresh", domain.ModuleAvatar, "清新自然", "leaf", "", "1:1·户外", 3),
		cat("c_illust", domain.ModuleAvatar, "插画风", "paint", "", "1:1·插画", 4),
	}
	if err := up(&cats); err != nil {
		return err
	}

	gen := func(prompt string, w, h int, style string, post []domain.PostOp) domain.JSON {
		return domain.MustJSON(domain.GenConfig{Engine: "genmodel", Mode: "reference", Prompt: prompt,
			NegativePrompt: "text, watermark, extra fingers, distorted face", Strength: 0.55,
			Output: &domain.GenOutput{Width: w, Height: h}, IdentityCheck: style != "illustration", Style: style, Post: post})
	}
	tpl := func(id string, m domain.Module, name, cover, catID, subtitle, prompt string, tags []string, style string, sort int) domain.Template {
		w, h := 1200, 1600
		var post []domain.PostOp
		if m == domain.ModuleAvatar {
			w, h = 1024, 1024
			post = []domain.PostOp{{Op: "square_crop", Width: 1024}}
		}
		return domain.Template{ID: id, Module: m, CategoryID: str(catID), Name: name, Subtitle: str(subtitle), CoverKey: asset(cover),
			SampleKeys: domain.MustJSON([]string{asset(cover)}), CreditCost: 1, GenConfig: gen(prompt, w, h, style, post),
			Tags: domain.MustJSON(tags), IsHot: contains(tags, "热门"), Sort: sort, Status: 1, CreatedAt: now, UpdatedAt: now}
	}
	tpls := []domain.Template{
		tpl("t_interview", domain.ModulePro, "面试职业照", "pro_interview", "c_interview", "深蓝西装 · 半身 · 3:4", "专业半身职业照，深蓝西装、白衬衫，无领带，浅蓝灰摄影棚，柔和光线，保留本人特征，{gender}", []string{"热门"}, "photo", 1),
		tpl("t_business", domain.ModulePro, "商务精英", "pro_business", "c_business", "白色西装 · 办公室背景 · 3:4", "business portrait, white blazer, modern office background, confident, {gender}", []string{"热门"}, "photo", 2),
		tpl("t_lecturer", domain.ModulePro, "讲师介绍图", "pro_lecturer", "c_lecturer", "半身 · 浅灰背景 · 3:4", "half-body lecturer portrait, light grey background, warm smile, {gender}", nil, "photo", 3),
		tpl("t_website", domain.ModulePro, "官网头像", "pro_website", "c_website", "深灰西装 · 半身 · 3:4", "企业官网半身职业照，深灰西装，浅灰背景，自然表情，保留本人特征，{gender}", nil, "photo", 4),
		tpl("t_consultant", domain.ModulePro, "顾问形象照", "pro_consultant", "c_consultant", "深蓝西装 · 浅灰背景 · 3:4", "顾问半身形象照，深蓝西装，浅灰办公室背景，保留本人特征，{gender}", []string{"上新"}, "photo", 5),
		tpl("t_doctor", domain.ModulePro, "医生形象照", "pro_doctor", "c_doctor", "白大褂 · 浅灰背景 · 3:4", "医生半身形象照，白大褂，浅灰背景，保留本人特征，{gender}", nil, "photo", 6),

		tpl("t_autumn", domain.ModulePortrait, "秋日写真", "portrait_autumn", "c_autumn", "暖色调 · 户外 · 3:4", "autumn outdoor portrait, warm tones, golden hour, {gender}", []string{"热门"}, "photo", 1),
		tpl("t_street", domain.ModulePortrait, "法式街拍", "portrait_street", "c_french", "街拍 · 胶片感 · 3:4", "法式街拍，黑色西装外套，浅灰石墙街角，自然回眸，胶片色调，保留本人特征，{gender}", nil, "photo", 2),
		tpl("t_chinese", domain.ModulePortrait, "新中式写真", "portrait_chinese2", "c_chinese", "国风 · 庭院 · 3:4", "新中式半身写真，月白色丝质立领上衣，团扇，竹影庭院，自然光，保留本人特征，{gender}", []string{"热门"}, "photo", 3),
		tpl("t_birthday", domain.ModulePortrait, "清新生日照", "portrait_birthday2", "c_birthday", "气球 · 蛋糕 · 3:4", "birthday portrait with balloons and cake, pastel, {gender}", []string{"上新"}, "photo", 4),
		tpl("t_korean", domain.ModulePortrait, "韩系清透写真", "portrait_korean", "c_korean", "自然光 · 户外 · 3:4", "korean clean portrait, natural light, outdoor, {gender}", nil, "photo", 5),
		tpl("t_french", domain.ModulePortrait, "法式氛围写真", "portrait_french", "c_french", "窗边 · 胶片感 · 3:4", "法式氛围写真，针织衫，咖啡馆窗边坐姿，柔和自然光，保留本人特征，{gender}", nil, "photo", 6),

		tpl("t_av_premium", domain.ModuleAvatar, "微信高级感", "avatar_premium", "c_premium", "影棚光 · 1:1", "premium studio avatar, soft studio light, {gender}", []string{"热门"}, "photo", 1),
		tpl("t_av_mood", domain.ModuleAvatar, "氛围感头像", "avatar_mood", "c_wechat", "暗调 · 1:1", "moody cinematic avatar, dark tones, {gender}", nil, "photo", 2),
		tpl("t_av_illust", domain.ModuleAvatar, "插画头像", "avatar_illust", "c_illust", "二次元 · 1:1", "anime illustration avatar of the same person, {gender}", []string{"上新"}, "illustration", 3),
		tpl("t_av_fresh", domain.ModuleAvatar, "清新自然头像", "avatar_fresh", "c_fresh", "户外 · 1:1", "fresh natural outdoor avatar, {gender}", nil, "photo", 4),
	}
	if err := up(&tpls); err != nil {
		return err
	}

	cols := []domain.Collection{
		{ID: "col_light", Module: domain.ModulePortrait, Name: "自然光 · 轻写真", CoverKey: asset("collection_light"), Description: str("自然光，少修饰"), Sort: 1, Status: 1},
		{ID: "col_half", Module: domain.ModulePortrait, Name: "半身写真", CoverKey: asset("portrait_autumn"), Description: str("半身构图"), Sort: 3, Status: 1},
		{ID: "col_festival", Module: domain.ModulePortrait, Name: "节日主题", CoverKey: asset("portrait_birthday2"), Description: str("生日、圣诞、新年"), Sort: 4, Status: 1},
		{ID: "col_mood", Module: domain.ModulePortrait, Name: "城市里的电影感", CoverKey: asset("collection_mood"), Description: str("胶片感与暗调"), Sort: 2, Status: 1},
	}
	if err := up(&cols); err != nil {
		return err
	}
	var colT []domain.CollectionTemplate
	for i, id := range []string{"t_korean", "t_french", "t_autumn", "t_street"} {
		colT = append(colT, domain.CollectionTemplate{CollectionID: "col_light", TemplateID: id, Sort: i})
	}
	for i, id := range []string{"t_autumn", "t_chinese", "t_korean"} {
		colT = append(colT, domain.CollectionTemplate{CollectionID: "col_half", TemplateID: id, Sort: i})
	}
	for i, id := range []string{"t_birthday", "t_chinese"} {
		colT = append(colT, domain.CollectionTemplate{CollectionID: "col_festival", TemplateID: id, Sort: i})
	}
	for i, id := range []string{"t_street", "t_french", "t_autumn"} {
		colT = append(colT, domain.CollectionTemplate{CollectionID: "col_mood", TemplateID: id, Sort: i})
	}
	if err := up(&colT); err != nil {
		return err
	}

	link := domain.MustJSON(map[string]any{"type": "upload"})
	banners := []domain.Banner{
		{ID: "bn_idphoto", Module: domain.ModuleIDPhoto, Title: "制作标准\n证件照", Subtitle: str(""), ImageKey: asset("banner_id"), Link: link, Sort: 1, Status: 1},
		{ID: "bn_pro", Module: domain.ModulePro, Title: "你的下一张\n职业形象照", Subtitle: str(""), ImageKey: asset("banner_pro"), Link: link, Sort: 1, Status: 1},
		{ID: "bn_portrait", Module: domain.ModulePortrait, Title: "自然光写真", Subtitle: str(""), ImageKey: asset("banner_portrait"), Link: link, Sort: 1, Status: 1},
		{ID: "bn_avatar", Module: domain.ModuleAvatar, Title: "换一张\n专属头像", Subtitle: str(""), ImageKey: asset("banner_avatar"), Link: link, Sort: 1, Status: 1},
	}
	if err := up(&banners); err != nil {
		return err
	}

	// runtime config defaults (insert only if missing)
	for k, v := range config.Defaults {
		row := domain.AppConfig{Key: k, Value: domain.MustJSON(v), UpdatedAt: now}
		if err := db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
	}

	// admin user
	if adminUser != "" {
		var n int64
		db.WithContext(ctx).Model(&domain.AdminUser{}).Where("username = ?", adminUser).Count(&n)
		if n == 0 {
			hash, _ := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
			if err := db.WithContext(ctx).Create(&domain.AdminUser{ID: "adm_" + fmt.Sprintf("%022d", 1), Username: adminUser, PasswordHash: string(hash), Role: "admin", Status: 1, CreatedAt: now}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func uploadAssets(ctx context.Context, store storage.ObjectStore, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("seed assets: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jpg" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		key := "assets/seed/" + e.Name()
		if err := store.Put(ctx, key, bytes.NewReader(data), int64(len(data)), "image/jpeg"); err != nil {
			return fmt.Errorf("upload %s: %w", key, err)
		}
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
