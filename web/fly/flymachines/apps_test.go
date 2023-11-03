package flymachines

import (
	"context"
	"net/http"
	"os"
	"testing"

	_ "github.com/joho/godotenv/autoload"
	"within.website/x/misc/namegen"
)

func TestAppLifecycle(t *testing.T) {
	token, ok := os.LookupEnv("FLY_API_TOKEN")
	if !ok {
		t.Skip("no FLY_API_TOKEN")
	}

	cli := New(token, http.DefaultClient)
	ctx := context.Background()

	name := namegen.Next()
	t.Logf("creating app %s", name)

	_, err := cli.CreateApp(ctx, CreateAppArgs{
		AppName: name,
		Network: "xtest",
		OrgSlug: "personal",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("created app %s", name)

	t.Logf("getting app %s", name)
	app, err := cli.GetApp(ctx, name)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("got app %s: %s", app.Name, app.Status)

	t.Log("listing all apps")

	apps, err := cli.GetApps(ctx, "personal")
	if err != nil {
		t.Fatal(err)
	}

	for _, app := range apps {
		t.Logf("app: %s", app.Name)
	}

	t.Logf("deleting app %s", name)
	if err := cli.DeleteApp(ctx, name); err != nil {
		t.Fatal(err)
	}
}
