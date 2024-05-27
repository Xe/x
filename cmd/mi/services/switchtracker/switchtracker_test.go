package switchtracker_test

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"within.website/x/cmd/mi/models"
	"within.website/x/cmd/mi/services/switchtracker"
	pb "within.website/x/proto/mi"
)

var (
	//go:embed testdata/members.json
	membersJSON []byte
)

func TestSwitch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir := t.TempDir()
	dao, err := models.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create dao: %v", err)
	}

	st := switchtracker.New(dao)

	// Import members and create root switch

	var members []*models.Member
	if err := json.Unmarshal(membersJSON, &members); err != nil {
		t.Fatalf("failed to unmarshal members: %v", err)
	}

	for _, m := range members {
		if err := dao.DB().Create(m).Error; err != nil {
			t.Fatalf("failed to create member: %v", err)
		}
	}

	if err := dao.DB().Create(&models.Switch{
		ID:       ulid.MustNew(ulid.Now(), rand.Reader).String(),
		MemberID: members[0].ID,
	}).Error; err != nil {
		t.Fatalf("failed to create root switch: %v", err)
	}

	resp, err := st.Members(ctx, nil)
	if err != nil {
		t.Errorf("failed to get members: %v", err)
	}

	if len(resp.Members) != len(members) {
		t.Errorf("expected %d members, got %d", len(members), len(resp.Members))
	}

	front, err := st.WhoIsFront(ctx, nil)
	if err != nil {
		t.Errorf("failed to get front: %v", err)
	}

	if front.Member.Name != members[0].Name {
		t.Errorf("expected front to be %s, got %s", members[0].Name, front.Member.Name)
	}

	_, err = st.Switch(ctx, &pb.SwitchReq{MemberName: members[1].Name})
	if err != nil {
		t.Errorf("failed to switch front: %v", err)
	}

	t.Log("trying to switch to current front")
	front, err = st.WhoIsFront(ctx, nil)
	if err != nil {
		t.Errorf("failed to get front: %v", err)
	}

	if front.Member.Name != members[1].Name {
		t.Errorf("expected front to be %s, got %s", members[1].Name, front.Member.Name)
	}

	front, err = st.WhoIsFront(ctx, nil)
	if err != nil {
		t.Errorf("failed to get front: %v", err)
	}

	_, err = st.Switch(ctx, &pb.SwitchReq{MemberName: front.Member.Name})
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if !strings.HasSuffix(err.Error(), "cannot switch to yourself") {
		t.Errorf("expected error to be 'cannot switch to yourself', got %v", err)
	}

	switches, err := st.ListSwitches(ctx, &pb.ListSwitchesReq{
		Count: 10,
	})
	if err != nil {
		t.Errorf("failed to list switches: %v", err)
	}

	for _, s := range switches.Switches {
		s := s
		t.Run("get switch "+s.GetSwitch().GetId(), func(t *testing.T) {
			fc, err := st.GetSwitch(ctx, &pb.GetSwitchReq{
				Id: s.GetSwitch().GetId(),
			})
			if err != nil {
				t.Errorf("failed to get switch: %v", err)
			}

			if fc.GetSwitch().GetId() != s.GetSwitch().GetId() {
				t.Errorf("expected switch ID to be %s, got %s", s.GetSwitch().GetId(), fc.GetSwitch().GetId())
			}
		})
	}
}
