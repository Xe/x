// Command iamd-openapi derives standard OAuth HTTP documentation from the same
// descriptors as Vanguard. The Connect OpenAPI generator documents RPC JSON;
// OAuth wire encodings and raw HttpBody responses need this additional pass.
package main

import (
	"fmt"
	"os"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
	oidcv1 "within.website/x/gen/within/website/x/iam/oidc/v1"
	"within.website/x/internal"
)

type object = map[string]any

func schemaObject() object                       { return object{"type": "object", "additionalProperties": true} }
func content(media string, schema object) object { return object{media: object{"schema": schema}} }
func response(description, media string, schema object) object {
	return object{"description": description, "content": content(media, schema)}
}
func run() error {
	data, err := os.ReadFile("docs/api/iamd-oidc-connect.openapi.yaml")
	if err != nil {
		return err
	}
	doc := object{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	paths := object{}
	services := oidcv1.File_within_website_x_iam_oidc_v1_oidc_proto.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			method := service.Methods().Get(j)
			rule, ok := proto.GetExtension(method.Options(), annotations.E_Http).(*annotations.HttpRule)
			if !ok {
				return fmt.Errorf("missing route for %s", method.FullName())
			}
			rules := append([]*annotations.HttpRule{rule}, rule.GetAdditionalBindings()...)
			for _, binding := range rules {
				verb, path := "get", binding.GetGet()
				if path == "" {
					verb, path = "post", binding.GetPost()
				}
				if path == "" {
					return fmt.Errorf("unsupported route for %s", method.FullName())
				}
				operation := object{"operationId": strings.ReplaceAll(string(method.FullName()), ".", "_") + "_" + verb, "summary": string(method.Name()), "tags": []string{string(service.Name())}}
				if verb == "get" {
					parameters := []any{}
					fields := method.Input().Fields()
					for k := 0; k < fields.Len(); k++ {
						field := fields.Get(k)
						if field.Kind() != protoreflect.StringKind {
							continue
						}
						parameters = append(parameters, object{"name": string(field.Name()), "in": "query", "schema": object{"type": "string"}})
					}
					if len(parameters) > 0 {
						operation["parameters"] = parameters
					}
				} else {
					media := "application/x-www-form-urlencoded"
					schema := object{"$ref": "#/components/schemas/within.website.x.iam.oidc.v1.ProtocolRequest"}
					if binding.GetBody() == "document" {
						media = "application/json"
						schema = schemaObject()
					}
					operation["requestBody"] = object{"required": true, "content": content(media, schema)}
				}
				errorSchema := object{"type": "object", "required": []string{"error"}, "properties": object{"error": object{"type": "string"}, "error_description": object{"type": "string"}}}
				responses := object{"400": response("OAuth protocol error", "application/json", errorSchema), "500": response("Server error", "application/json", errorSchema)}
				browser := string(service.Name()) == "BrowserService" || path == "/authorize"
				if browser && path != "/upstream/backchannel-logout" {
					responses["200"] = response("Browser page (Connect callers receive structured JSON)", "text/html", object{"type": "string"})
					for _, status := range []string{"302", "303"} {
						responses[status] = object{"description": "Browser redirect", "headers": object{"Location": object{"schema": object{"type": "string", "format": "uri-reference"}}}}
					}
				} else {
					status := "200"
					if path == "/register" || path == "/admin/identities" {
						status = "201"
					}
					responses[status] = response("Protocol response", "application/json", schemaObject())
				}
				if path == "/token" {
					responses["401"] = response("Client authentication failed", "application/json", errorSchema)
				}
				if path == "/userinfo" || strings.HasPrefix(path, "/admin/") {
					responses["401"] = response("Authentication required", "application/json", errorSchema)
					responses["403"] = response("Access denied", "application/json", errorSchema)
				}
				if path == "/register" {
					responses["429"] = response("Registration rate limit", "application/json", errorSchema)
				}
				operation["responses"] = responses
				item, _ := paths[path].(object)
				if item == nil {
					item = object{}
					paths[path] = item
				}
				item[verb] = operation
			}
		}
	}
	doc["paths"] = paths
	doc["servers"] = []any{object{"url": "https://iamd.xeserv.us"}}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile("docs/api/iamd-oidc.openapi.yaml", out, 0644)
}
func main() {
	internal.HandleStartup()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
