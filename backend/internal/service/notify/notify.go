// Package notify sends WeChat subscribe messages (docs/DECISIONS.md D-07).
package notify

import (
	"context"
	"log/slog"

	"yingji/backend/internal/domain"
	"yingji/backend/internal/provider/wechat"
	"yingji/backend/internal/service/auth"
)

type Service struct {
	wx       *wechat.Client
	auth     *auth.Service
	tmplTask string
	log      *slog.Logger
}

func New(wx *wechat.Client, a *auth.Service, taskTemplateID string, log *slog.Logger) *Service {
	return &Service{wx: wx, auth: a, tmplTask: taskTemplateID, log: log}
}

// TaskFinished notifies the task owner when they accepted the subscription.
func (s *Service) TaskFinished(ctx context.Context, t *domain.Task, targetName string) error {
	if !t.NotifyRequested || s.tmplTask == "" || !s.wx.Configured() {
		return nil
	}
	openid, err := s.auth.OpenID(ctx, t.UserID)
	if err != nil {
		return nil
	}
	status := "生成成功"
	page := "pages/result/index?task_id=" + t.ID
	if t.WorkID != nil {
		page += "&work_id=" + *t.WorkID
	}
	if t.Status == domain.TaskFailed {
		status = "生成失败，次数已返还"
		page = "pages-mine/records/index"
	}
	data := map[string]string{"thing1": targetName, "phrase2": status}
	if err := s.wx.SendSubscribeMessage(ctx, openid, s.tmplTask, page, data); err != nil {
		s.log.Warn("subscribe message failed", "task", t.ID, "err", err)
	}
	return nil
}
