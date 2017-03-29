package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Bowery/prompt"
	jwtcreds "github.com/Xe/tools/svc/credentials/jwt"
	svc "github.com/Xe/tools/svc/proto"
	"github.com/Xe/uuid"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/joho/godotenv"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	kingpin "gopkg.in/alecthomas/kingpin.v1"
)

var (
	app     = kingpin.New("svc", "A simple service manager")
	debug   = app.Flag("debug", "print debugging logs?").Bool()
	host    = app.Flag("host", "host to do these changes to").String()
	dataDir = app.Flag("data-dir", "place for svc to store data").Default(filepath.Join(os.Getenv("HOME"), ".local/within/svc")).String()

	create            = app.Command("create", "Create a new application")
	createName        = create.Flag("name", "name of the application").Required().String()
	createEnvFile     = create.Flag("env-file", "file with key->value envvars").ExistingFile()
	createEnvironment = create.Flag("env", "environment variables for the program").StringMap()
	createLabels      = create.Flag("label", "additional labels to attach to the service").StringMap()
	//createAuthorizedUsers = create.Flag("authorized-user", "additional user to allow modification access to").Strings()
	//createExclusive       = create.Flag("exclusive", "can this only ever have one copy running at once?").Bool()
	//createInstances       = create.Flag("instances", "number of instances of the backend service").Default("1").Int()
	createDockerImage = create.Arg("docker image", "docker image to execute for this service").Required().String()

	createToken          = app.Command("create-token", "Creates the initial server control token")
	createTokenJwtSecret = createToken.Flag("jwt-secret", "jwt secret used on the server").Required().String()
	createTokenUsername  = createToken.Arg("username", "username to create token for").Required().String()

	deleteCmd  = app.Command("delete", "Deletes an application by name")
	deleteName = deleteCmd.Arg("name", "name of the service").Required().String()

	hostCmd       = app.Command("host", "Host management")
	hostAdd       = hostCmd.Command("add", "Add a host to the state file")
	hostAddTor    = hostAdd.Flag("tor", "connect to this over tor?").Bool()
	hostAddCaCert = hostAdd.Flag("ca-cert", "ca certificate of the server").Default("ca.pem").File()
	hostAddCert   = hostAdd.Flag("cert", "client certificate").Default("cert.pem").File()
	hostAddKey    = hostAdd.Flag("key", "client ssl key").Default("key.pem").File()
	hostAddName   = hostAdd.Arg("name", "name of host to add").Required().String()
	hostAddAddr   = hostAdd.Arg("addr", "address of taget server (host:port)").Required().String()

	hostRemove     = hostCmd.Command("remove", "Remove a host from the state file")
	hostRemoveName = hostRemove.Arg("name", "name of host to remove").Required().String()

	inspect     = app.Command("inspect", "Inspect an application")
	inspectName = inspect.Arg("name", "name of the service").String()

	list           = app.Command("list", "List apps running with this backend")
	listLabelKey   = list.Flag("labelKey", "label key to match for").String()
	listLabelValue = list.Flag("labelValue", "label value to match for (with labelKey)").String()

	update            = app.Command("update", "Update an application")
	updateImage       = update.Flag("image", "new docker image to use for this service").String()
	updateEnvAdd      = update.Flag("env-add", "new environment variables to set").StringMap()
	updateEnvRm       = update.Flag("env-rm", "environment variables to remove").Strings()
	updateGrantUsers  = update.Flag("grant-user", "grant a user permission to this service").Strings()
	updateRevokeUsers = update.Flag("revoke-user", "revoke a user's permission to this service").Strings()
	updateName        = update.Flag("name", "name of the service to update").Required().String()
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

		caCertData, err := ioutil.ReadAll(*hostAddCaCert)
		if err != nil {
			log.Fatal(err)
		}

		clientCertData, err := ioutil.ReadAll(*hostAddCert)
		if err != nil {
			log.Fatal(err)
		}

		clientKeyData, err := ioutil.ReadAll(*hostAddKey)
		if err != nil {
			log.Fatal(err)
		}

		h := &Host{
			Name:   *hostAddName,
			Addr:   *hostAddAddr,
			Token:  token,
			Tor:    *hostAddTor,
			CaCert: caCertData,
			Cert:   clientCertData,
			Key:    clientKeyData,
		}

		state.Hosts[h.Name] = h
		writeState(state)

		log.Println("Host added to hosts file.")
		return
	case "host remove":
		_, exists := state.Hosts[*hostRemoveName]
		if !exists {
			log.Fatalf("no such host %q", *hostRemoveName)
		}

		delete(state.Hosts, *hostRemoveName)
		writeState(state)

		log.Printf("Host %q removed from hosts file", *hostRemoveName)
		return
	case "create-token":
		now := time.Now()

		hostname, _ := os.Hostname()
		tid := uuid.New()

		token := jwt.NewWithClaims(jwt.SigningMethodHS512, &jwt.StandardClaims{
			IssuedAt: now.Unix(),
			Issuer:   hostname,
			Subject:  *createTokenUsername,
			Id:       tid,
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

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(hostInfo.CaCert)

	connCreds := credentials.NewTLS(&tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	})

	creds := jwtcreds.NewFromToken(hostInfo.Token)
	conn, err := grpc.Dial(hostInfo.Addr,
		grpc.WithTransportCredentials(connCreds),
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
		env := map[string]string{}

		for key, val := range *createEnvironment {
			env[key] = val
		}

		if *createEnvFile != "" {
			emap, err := godotenv.Read(*createEnvFile)
			if err != nil {
				log.Fatal(err)
			}

			for key, val := range emap {
				env[key] = val
			}
		}

		m := &svc.Manifest{
			DockerImage: *createDockerImage,
			Environment: env,
			Labels:      *createLabels,
			Name:        *createName,
		}

		app, err := c.Create(context.Background(), m)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("%s created", app.Name)
		return
	case update.FullCommand():
		_, err := c.Update(context.Background(), &svc.AppUpdate{
			Name:        *updateName,
			NewImage:    *updateImage,
			EnvAdd:      *updateEnvAdd,
			EnvRm:       *updateEnvRm,
			GrantUsers:  *updateGrantUsers,
			RevokeUsers: *updateRevokeUsers,
		})
		if err != nil {
			log.Fatal(err)
		}

		log.Println("success")
		return
	case inspect.FullCommand():
		app, err := c.Inspect(context.Background(), &svc.AppInspect{
			Name: *inspectName,
		})
		if err != nil {
			log.Fatal(err)
		}

		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		e.Encode(app)
		return
	case deleteCmd.FullCommand():
		ok, err := c.Delete(context.Background(), &svc.AppDelete{Name: *deleteName})
		if err != nil {
			log.Fatal(err)
		}

		log.Println(ok.Message)
	}
}

type state struct {
	Hosts map[string]*Host
}

type Host struct {
	Name   string
	Addr   string
	Token  string
	Tor    bool
	CaCert []byte
	Cert   []byte
	Key    []byte
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
