package miv1

import "log/slog"

func (m *Member) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("id", int(m.GetId())),
		slog.String("name", m.GetName()),
		slog.String("avatar_url", m.GetAvatarUrl()),
	)
}

func (s *Switch) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", s.GetId()),
		slog.Int("member_id", int(s.GetMemberId())),
		slog.String("started_at", s.GetStartedAt()),
		slog.String("ended_at", s.GetEndedAt()),
	)
}

func (sr *SwitchReq) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("member_name", sr.GetMemberName()),
	)
}

func (sr *SwitchResp) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("old", sr.GetOld()),
		slog.Any("current", sr.GetCurrent()),
	)
}
