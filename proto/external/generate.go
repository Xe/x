package external

//go:generate protoc --proto_path=. --proto_path=.. --go_out=./jsonfeed --go_opt=paths=source_relative ./jsonfeed.proto
