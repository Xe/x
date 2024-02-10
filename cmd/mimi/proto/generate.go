package scheduling

func init() {}

//go:generate protoc --proto_path=. --go_out=../modules/scheduling --go_opt=paths=source_relative --go-grpc_out=../modules/scheduling --go-grpc_opt=paths=source_relative --twirp_out=../modules/scheduling --twirp_opt=paths=source_relative scheduling.proto
