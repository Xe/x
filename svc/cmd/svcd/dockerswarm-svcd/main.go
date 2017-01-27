package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	svc "github.com/Xe/tools/svc/proto"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/facebookgo/flagenv"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

var (
	listenAddress = flag.String("listen", "127.0.0.1:23142", "tcp host:port to listen on")
	sslCert       = flag.String("tls-cert", "cert.pem", "tls certificate to read from")
	sslKey        = flag.String("tls-key", "key.pem", "tls private key")
	caCert        = flag.String("ca-cert", "ca.pem", "ca public cert")
	jwtSecret     = flag.String("jwt-secret", "hunter2", "secret used to sign jwt's")
)

const admin = "xena"

type server struct {
	docker *client.Client

	sync.Mutex
	state map[string][]string
}

func (s *server) LoadState(fname string) error {
	s.Lock()
	defer s.Unlock()

	fin, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer fin.Close()

	return json.NewDecoder(fin).Decode(&s.state)
}

func (s *server) SaveState(fname string) error {
	s.Lock()
	defer s.Unlock()

	fout, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer fout.Close()

	return json.NewEncoder(fout).Encode(&s.state)
}

func (s *server) List(ctx context.Context, params *svc.AppsListParams) (*svc.AppsList, error) {
	user, err := s.checkAuth(ctx)
	if err != nil {
		return nil, err
	}

	svcs, err := s.docker.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, err
	}

	result := &svc.AppsList{}

	for _, ssvc := range svcs {
		env := func(kv []string) map[string]string {
			result := map[string]string{}

			for _, pair := range kv {
				split := strings.SplitN(pair, "=", 2)
				result[split[0]] = split[1]
			}

			return result
		}(ssvc.Spec.TaskTemplate.ContainerSpec.Env)

		au := s.state[ssvc.Spec.Name]
		if au == nil {
			s.state[ssvc.Spec.Name] = []string{admin}
			s.SaveState("state.json")
		}

		allowed := false

		if user == admin {
			allowed = true
		}

		for _, allowedUser := range au {
			if user == allowedUser {
				allowed = true
			}
		}

		if !allowed {
			continue
		}

		result.Apps = append(result.Apps, &svc.App{
			Id:              ssvc.ID,
			Name:            ssvc.Spec.Name,
			DockerImage:     ssvc.Spec.TaskTemplate.ContainerSpec.Image,
			Environment:     env,
			Labels:          ssvc.Spec.Labels,
			AuthorizedUsers: au,
		})
	}

	return result, nil
}

func (s *server) Create(ctx context.Context, manifest *svc.Manifest) (*svc.App, error) {
	user, err := s.checkAuth(ctx)
	if err != nil {
		return nil, err
	}

	env := []string{}

	for key, val := range manifest.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}

	spec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name:   manifest.Name,
			Labels: manifest.Labels,
		},

		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image: manifest.DockerImage,
				Env:   env,
			},
		},

		Mode: swarm.ServiceMode{
			Replicated: &swarm.ReplicatedService{},
		},
	}

	resp, err := s.docker.ServiceCreate(ctx, spec, types.ServiceCreateOptions{})
	if err != nil {
		return nil, err
	}

	ssvc, _, err := s.docker.ServiceInspectWithRaw(ctx, resp.ID)
	if err != nil {
		return nil, err
	}

	app := &svc.App{
		Id:              ssvc.ID,
		Name:            ssvc.Spec.Name,
		DockerImage:     ssvc.Spec.TaskTemplate.ContainerSpec.Image,
		Environment:     manifest.Environment,
		Labels:          ssvc.Spec.Labels,
		AuthorizedUsers: []string{user},
	}

	return app, nil
}

func (s *server) Update(ctx context.Context, params *svc.AppUpdate) (*svc.App, error) {
	return nil, errors.New("not implemented")
}

func (s *server) Inspect(ctx context.Context, params *svc.AppInspect) (*svc.App, error) {
	return nil, errors.New("not implemented")
}

func (s *server) Delete(ctx context.Context, params *svc.AppDelete) (*svc.Ok, error) {
	return nil, errors.New("not implemented")
}

func main() {
	flag.Parse()
	flagenv.Parse()

	cert, err := tls.LoadX509KeyPair(*sslCert, *sslKey)
	if err != nil {
		log.Fatal(err)
	}

	rawCaCert, err := ioutil.ReadFile(*caCert)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(rawCaCert)

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
	})

	gs := grpc.NewServer(grpc.Creds(creds))

	defaultHeaders := map[string]string{"User-Agent": "dockerswarm-svcd"}
	cli, err := client.NewClient(client.DefaultDockerHost, client.DefaultVersion, nil, defaultHeaders)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{
		docker: cli,
		state:  map[string][]string{},
	}

	err = s.LoadState("state.json")
	if err != nil {
		log.Fatal(err)
	}

	svc.RegisterAppsServer(gs, s)

	l, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		log.Fatal(err)
	}

	err = gs.Serve(l)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *server) checkAuth(ctx context.Context) (string, error) {
	var err error

	md, ok := metadata.FromContext(ctx)
	if !ok {
		return "", grpc.Errorf(codes.Unauthenticated, "valid token required.")
	}

	jwtToken, ok := md["authorization"]
	if !ok {
		return "", grpc.Errorf(codes.Unauthenticated, "valid token required.")
	}

	clms := &jwt.StandardClaims{}

	p := &jwt.Parser{}
	_, err = p.ParseWithClaims(jwtToken[0], clms, jwt.Keyfunc(func(t *jwt.Token) (interface{}, error) {
		return []byte(*jwtSecret), nil
	}))
	if err != nil {
		log.Printf("rpc error: %v", err)
		return "", grpc.Errorf(codes.Unauthenticated, "valid token requried.")
	}

	return clms.Subject, nil
}
