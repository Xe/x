package mimi

//go:generate protoc --proto_path=. --proto_path=.. --go_out=./announce --go_opt=paths=source_relative --go-grpc_out=./announce --go-grpc_opt=paths=source_relative --twirp_out=./announce --twirp_opt=paths=source_relative ./announce.proto
//go:generate protoc --proto_path=. --proto_path=.. --go_out=./statuspage --go_opt=paths=source_relative --go-grpc_out=./statuspage --go-grpc_opt=paths=source_relative  --twirp_out=./statuspage --twirp_opt=paths=source_relative ./statuspage.proto
