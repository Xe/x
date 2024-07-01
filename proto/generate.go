package proto

//go:generate protoc --proto_path=. --go_out=./mi --go_opt=paths=source_relative --twirp_out=./mi --twirp_opt=paths=source_relative ./mi.proto
//go:generate protoc --proto_path=. --go_out=./sanguisuga --go_opt=paths=source_relative --twirp_out=./sanguisuga --twirp_opt=paths=source_relative ./sanguisuga.proto
//go:generate protoc --proto_path=. --go_out=./uplodr --go_opt=paths=source_relative --go-grpc_out=./uplodr --go-grpc_opt=paths=source_relative ./uplodr.proto
