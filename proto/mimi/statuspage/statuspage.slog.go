package statuspage

import "log/slog"

func (m *Meta) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("unsubscribe", m.GetUnsubscribe()),
		slog.String("documentation", m.GetDocumentation()),
	)
}

func (p *Page) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", p.GetId()),
		slog.String("status_indicator", p.GetStatusIndicator()),
		slog.String("status_description", p.GetStatusDescription()),
	)
}

func (cu *ComponentUpdate) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("created_at", cu.GetCreatedAt()),
		slog.String("new_status", cu.GetNewStatus()),
		slog.String("old_status", cu.GetOldStatus()),
		slog.String("id", cu.GetId()),
		slog.String("component_id", cu.GetComponentId()),
	)
}

func (c *Component) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("created_at", c.GetCreatedAt()),
		slog.String("id", c.GetId()),
		slog.String("name", c.GetName()),
		slog.String("status", c.GetStatus()),
	)
}

func (iu *IncidentUpdate) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("body", iu.GetBody()),
		slog.String("created_at", iu.GetCreatedAt()),
		slog.String("status", iu.GetStatus()),
		slog.String("updated_at", iu.GetUpdatedAt()),
		slog.String("id", iu.GetId()),
		slog.String("incident_id", iu.GetIncidentId()),
	)
}

func (i *Incident) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("backfilled", i.GetBackfilled()),
		slog.String("created_at", i.GetCreatedAt()),
		slog.String("impact", i.GetImpact()),
		slog.String("monitoring_at", i.GetMonitoringAt()),
		slog.String("postmortem_body", i.GetPostmortemBody()),
		slog.String("postmortem_published_at", i.GetPostmortemPublishedAt()),
		slog.String("resolved_at", i.GetResolvedAt()),
		slog.String("shortlink", i.GetShortlink()),
		slog.String("status", i.GetStatus()),
		slog.String("updated_at", i.GetUpdatedAt()),
		slog.String("id", i.GetId()),
		slog.Group("incident_updates", "children", i.GetIncidentUpdates()),
		slog.String("name", i.GetName()),
	)
}
