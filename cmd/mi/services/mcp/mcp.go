package mcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"within.website/x"
	miv1 "within.website/x/gen/within/website/x/mi/v1"
)

type Server struct {
	st miv1.SwitchTracker
	es miv1.Events
}

type switchReq struct {
	Name string `json:"name" jsonschema:"The member to switch to"`
}

type switchResp struct {
	// OldMember      string `json:"oldMember" jsonschema:"The member that just switched out"`
	// SwitchDuration string `json:"switchDuration" jsonschema:"How long the last switch lasted"`
}

func (s *Server) switchFront(ctx context.Context, req *mcp.CallToolRequest, sr switchReq) (*mcp.CallToolResult, *switchResp, error) {
	resp, err := s.st.Switch(ctx, &miv1.SwitchReq{
		MemberName: sr.Name,
	})
	if err != nil {
		return nil, nil, err
	}
	_ = resp

	result := &switchResp{
		// OldMember: resp.,
	}

	return nil, result, nil
}

type whoIsFrontReq struct{}

type whoIsFrontResp struct {
	Name      string `json:"name" jsonschema:"The member name"`
	AvatarURL string `json:"avatarURL" jsonschema:"The member's avatar URL"`
	StartedAt string `json:"startedAt" jsonschema:"When the switch started"`
}

func (s *Server) whoIsFront(ctx context.Context, req *mcp.CallToolRequest, wif whoIsFrontReq) (*mcp.CallToolResult, *whoIsFrontResp, error) {
	resp, err := s.st.WhoIsFront(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, nil, err
	}

	result := &whoIsFrontResp{
		Name:      resp.Member.Name,
		AvatarURL: resp.Member.AvatarUrl,
		StartedAt: resp.Switch.StartedAt,
	}

	return nil, result, nil
}

type listSystemMembersReq struct{}

type listSystemMembersResp struct {
	Message string   `json:"message,omitempty"`
	Members []string `json:"members,omitempty"`
}

func (s *Server) listSystemMembers(ctx context.Context, req *mcp.CallToolRequest, lsm listSystemMembersReq) (*mcp.CallToolResult, *listSystemMembersResp, error) {
	resp, err := s.st.Members(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, nil, err
	}

	if len(resp.Members) == 0 {
		emptyResult := "# System Members\n\nNo members found in the system."
		return nil, &listSystemMembersResp{Message: emptyResult}, nil
	}

	result := &listSystemMembersResp{}

	for _, member := range resp.Members {
		result.Members = append(result.Members, member.GetName())
	}

	return nil, result, nil
}

const dateLayout = "2006-01-02"

type listEventsReq struct{}

type listEventsResp struct {
	Events []EventItem `json:"events,omitempty"`
}

type EventItem struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

func (s *Server) listEvents(ctx context.Context, req *mcp.CallToolRequest, _ listEventsReq) (*mcp.CallToolResult, *listEventsResp, error) {
	resp, err := s.es.Get(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, nil, err
	}

	result := &listEventsResp{}
	for _, ev := range resp.Events {
		result.Events = append(result.Events, EventItem{
			Name:        ev.Name,
			URL:         ev.Url,
			StartDate:   ev.StartDate.AsTime().Format(dateLayout),
			EndDate:     ev.EndDate.AsTime().Format(dateLayout),
			Location:    ev.Location,
			Description: ev.Description,
		})
	}

	return nil, result, nil
}

type addEventReq struct {
	Name        string `json:"name" jsonschema:"Name of the event"`
	URL         string `json:"url" jsonschema:"URL for the event"`
	StartDate   string `json:"startDate" jsonschema:"Start date in YYYY-MM-DD format"`
	EndDate     string `json:"endDate,omitempty" jsonschema:"End date in YYYY-MM-DD format, defaults to start date"`
	Location    string `json:"location" jsonschema:"Location of the event"`
	Description string `json:"description" jsonschema:"Description of the event"`
}

type addEventResp struct {
	Message string `json:"message"`
}

func (s *Server) addEvent(ctx context.Context, req *mcp.CallToolRequest, ae addEventReq) (*mcp.CallToolResult, *addEventResp, error) {
	startTime, err := time.Parse(dateLayout, ae.StartDate)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid startDate %q: %w", ae.StartDate, err)
	}

	endTime := startTime
	if ae.EndDate != "" {
		endTime, err = time.Parse(dateLayout, ae.EndDate)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid endDate %q: %w", ae.EndDate, err)
		}
	}

	_, err = s.es.Add(ctx, &miv1.Event{
		Name:        ae.Name,
		Url:         ae.URL,
		StartDate:   timestamppb.New(startTime),
		EndDate:     timestamppb.New(endTime),
		Location:    ae.Location,
		Description: ae.Description,
	})
	if err != nil {
		return nil, nil, err
	}

	return nil, &addEventResp{Message: "Event added successfully"}, nil
}

type removeEventReq struct {
	ID int32 `json:"id" jsonschema:"Event ID to remove"`
}

type removeEventResp struct {
	Message string `json:"message"`
}

func (s *Server) removeEvent(ctx context.Context, req *mcp.CallToolRequest, re removeEventReq) (*mcp.CallToolResult, *removeEventResp, error) {
	_, err := s.es.Remove(ctx, &miv1.Event{Id: re.ID})
	if err != nil {
		return nil, nil, err
	}

	return nil, &removeEventResp{Message: "Event removed successfully"}, nil
}

func New(st miv1.SwitchTracker, es miv1.Events) http.Handler {
	s := &Server{
		st: st,
		es: es,
	}

	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "mi",
		Version: x.Version,
		Title:   "Mi from Within",
	}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "switch-front",
		Description: "Record a switch in the database",
	}, s.switchFront)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "who-is-front",
		Description: "Find out who's front this time!",
	}, s.whoIsFront)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list-system-members",
		Description: "List all system members as Markdown",
	}, s.listSystemMembers)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list-events",
		Description: "List upcoming events",
	}, s.listEvents)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "add-event",
		Description: "Add an event to the feed",
	}, s.addEvent)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "remove-event",
		Description: "Remove an event from the feed",
	}, s.removeEvent)

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server { return srv }, nil)

	return handler
}
