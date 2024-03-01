package sanguisuga

import "log/slog"

func (s *Show) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("title", s.GetTitle()),
		slog.String("disk_path", s.GetDiskPath()),
		slog.String("quality", s.GetQuality()),
	)
}

func (tvs *TVSnatch) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("category", tvs.GetCategory()),
		slog.String("name", tvs.GetName()),
		slog.Bool("freeleech", tvs.GetFreeleech()),
		slog.String("torrent_id", tvs.GetTorrentId()),
	)
}

func (as *AnimeSnatch) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("fname", as.GetFname()),
		slog.String("show_name", as.GetShowName()),
		slog.String("episode", as.GetEpisode()),
		slog.String("resolution", as.GetResolution()),
		slog.String("crc32", as.GetCrc32()),
		slog.String("bot_name", as.GetBotName()),
		slog.String("pack_id", as.GetPackId()),
	)
}
