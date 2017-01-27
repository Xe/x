package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Bowery/prompt"
	jwtcreds "github.com/Xe/tools/svc/credentials/jwt"
	svc "github.com/Xe/tools/svc/proto"
	"github.com/Xe/uuid"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/grpc"
	kingpin "gopkg.in/alecthomas/kingpin.v1"
)

var (
	app     = kingpin.New("svc", "A simple service manager")
	debug   = app.Flag("debug", "print debugging logs?").Bool()
	host    = app.Flag("host", "host to do these changes to").String()
	dataDir = app.Flag("data-dir", "place for svc to store data").Default("/home/xena/.local/within/svc").String()

	createToken          = app.Command("create-token", "Creates the initial server control token")
	createTokenJwtSecret = createToken.Flag("jwt-secret", "jwt secret used on the server").Required().String()
	createTokenUsername  = createToken.Arg("username", "username to create token for").Required().String()

	list           = app.Command("list", "List apps running with this backend")
	listLabelKey   = list.Flag("labelKey", "label key to match for").String()
	listLabelValue = list.Flag("labelValue", "label value to match for (with labelKey)").String()

	create                = app.Command("create", "Create a new application")
	createName            = create.Flag("name", "name of the application").Required().String()
	createEnvFile         = create.Flag("env-file", "file with key->value envvars").String()
	createEnvironment     = create.Flag("env", "environment variables for the program").StringMap()
	createLabels          = create.Flag("label", "additional labels to attach to the service").StringMap()
	createAuthorizedUsers = create.Flag("authorized-user", "additional user to allow modification access to").Strings()
	createExclusive       = create.Flag("exclusive", "can this only ever have one copy running at once?").Bool()
	createInstances       = create.Flag("instances", "number of instances of the backend service").Default("1").Int()
	createDockerImage     = create.Arg("docker image", "docker image to execute for this service").Required().String()

	update              = app.Command("update", "Update an application")
	updateImage         = update.Flag("image", "new docker image to use for this service").String()
	updateEnvAdd        = update.Flag("env-add", "new environment variables to set").StringMap()
	updateEnvRm         = update.Flag("env-rm", "environment variables to remove").StringMap()
	updateLabelAdd      = update.Flag("label-add", "container labels to addB").StringMap()
	updateLabelRm       = update.Flag("label-rm", "container labels to remove").StringMap()
	updateGrantUsers    = update.Flag("grant-user", "grant a user permission to this service").Strings()
	updateRevokeUsers   = update.Flag("revoke-user", "revoke a user's permission to this service").Strings()
	updateInstanceCount = update.Flag("instances", "updates the instance count of the service").Int()

	inspect     = app.Command("inspect", "Inspect an application")
	inspectName = inspect.Arg("name", "name of the service").String()

	deleteCmd  = app.Command("delete", "Deletes an application by name")
	deleteName = deleteCmd.Arg("name", "name of the service").String()

	hostCmd        = app.Command("host", "Host management")
	hostAdd        = hostCmd.Command("add", "Add a host to the state file")
	hostAddTor     = hostAdd.Flag("tor", "connect to this over tor?").Bool()
	hostAddName    = hostAdd.Arg("name", "name of host to add").Required().String()
	hostAddAddr    = hostAdd.Arg("addr", "address of taget server (host:port)").Required().String()
	hostRemove     = hostCmd.Command("remove", "Remove a host from the state file")
	hostRemoveName = hostRemove.Arg("name", "name of host to remove").Required().String()
)

func main() {
	cmdline := kingpin.MustParse(app.Parse(os.Args[1:]))

	state, err := readState()
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Host file does not exist, please add a host with `svc host add`.")
		}

		log.Fatal(err)
	}
	writeState(state)

	switch cmdline {
	case "host add":
		token, err := prompt.Basic("token: ", true)
		if err != nil {
			log.Fatal(err)
		}

		h := &Host{
			Name:  *hostAddName,
			Addr:  *hostAddAddr,
			Token: token,
			Tor:   *hostAddTor,
		}

		state.Hosts[h.Name] = h
		writeState(state)

		log.Println("Host added to hosts file.")
		return
	case "host remove":
		log.Println("removing host not yet implemented")
		os.Exit(1)
	case "create-token":
		now := time.Now()
		nva := now.AddDate(0, 1, 0)  // Expiry time of this token
		nb4 := now.AddDate(0, 0, -1) // Not before then is this token valid

		hostname, _ := os.Hostname()
		tid := uuid.New()

		token := jwt.NewWithClaims(jwt.SigningMethodHS512, &jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			NotBefore: nb4.Unix(),
			ExpiresAt: nva.Unix(),
			Issuer:    hostname,
			Subject:   *createTokenUsername,
			Id:        tid,
		})

		tokenString, err := token.SignedString([]byte(*createTokenJwtSecret))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(tokenString)

		os.Exit(0)
	}

	if *host == "" {
		log.Fatal("--host must be supplied")
	}

	hostInfo, ok := state.Hosts[*host]
	if !ok {
		log.Fatalf("Requested host %q that doesn't exist in state", *host)
	}

	creds := jwtcreds.NewFromToken(hostInfo.Token)
	conn, err := grpc.Dial(hostInfo.Addr, grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(creds))
	if err != nil {
		log.Fatal(err)
	}

	c := svc.NewAppsClient(conn)

	// RPC commands
	switch cmdline {
	case list.FullCommand():
		apps, err := c.List(context.Background(), &svc.AppsListParams{})
		if err != nil {
			log.Fatal(err)
		}

		table := tablewriter.NewWriter(os.Stdout)

		table.SetHeader([]string{"ID", "Name", "Image", "Users"})

		for _, app := range apps.Apps {
			table.Append([]string{app.Id, app.Name, app.DockerImage, fmt.Sprintf("%v", app.AuthorizedUsers)})
		}
		table.Render()

	case create.FullCommand():
		log.Println("create not implemented")
	case update.FullCommand():
		log.Println("update not implemented")
	case inspect.FullCommand():
		log.Println("inspect not implemented")
	case deleteCmd.FullCommand():
		log.Println("delete not implemented")
	}
}

type state struct {
	Hosts map[string]*Host
}

type Host struct {
	Name  string
	Addr  string
	Token string
	Tor   bool
}

func readState() (*state, error) {
	s := &state{}

	fname := filepath.Join(*dataDir, "state.json")
	fin, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer fin.Close()

	err = json.NewDecoder(fin).Decode(s)

	return s, err
}

func writeState(s *state) error {
	fname := filepath.Join(*dataDir, "state.json")
	fout, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer fout.Close()

	return json.NewEncoder(fout).Encode(s)
}
