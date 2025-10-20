package mcp

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/emptypb"
	"within.website/x"
	miv1 "within.website/x/gen/within/website/x/mi/v1"
)

type Server struct {
	st miv1.SwitchTracker
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

func New(st miv1.SwitchTracker) http.Handler {
	s := &Server{
		st: st,
	}

	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "mi",
		Version: x.Version,
		Title:   "Mi from Within",
	}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "switch",
		Description: "Record a switch in the database",
	}, s.switchFront)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "who-is-front",
		Description: "Find out who's front this time!",
	}, s.whoIsFront)

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server { return srv }, nil)

	return handler
}
