package vastaicli

import (
	"context"
	"testing"
	"time"
)

func TestSearch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	instances, err := Search(ctx, "verified=False cuda_max_good>=12.1 gpu_ram>=12 num_gpus=1 inet_down>=850", "dph+")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("found %d candidates", len(instances))
}
