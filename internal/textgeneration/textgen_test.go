package textgen

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
)

func TestApplyCharacter(t *testing.T) {
	cr := new(ChatRequest)
	if err := cr.ApplyCharacter("yasomi"); err != nil {
		t.Fatalf("%v", err)
	}

	if cr.BotName != "Midori Yasomi" {
		t.Fatalf("expected bot name to be %q, got: %q", "Midori Yasomi", cr.BotName)
	}

	t.Log(cr.Context)
}

func TestApplyPreset(t *testing.T) {
	cr := new(ChatRequest)
	if err := cr.ApplyPreset("Kobold-Godlike"); err != nil {
		t.Fatalf("%v", err)
	}
}

func TestTextGen(t *testing.T) {
	if ok, _ := strconv.ParseBool(os.Getenv("TEXTGEN_REALWORLD")); !ok {
		t.Skip("TEXTGEN_REALWORLD is not set, not testing.")
	}

	cr := new(ChatRequest)
	if err := cr.ApplyPreset("Default"); err != nil {
		t.Fatalf("%v", err)
	}

	if err := cr.ApplyCharacter("yasomi"); err != nil {
		t.Fatalf("%v", err)
	}

	cr.Input = "So, what's the deal with airline food?"
	cr.MaxNewTokens = 200
	cr.DoSample = true
	cr.EarlyStopping = true

	resp, err := Generate(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(resp.Data[0])
}
