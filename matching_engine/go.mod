module github.com/Marwan051/tradding_platform_game/matching_engine

go 1.23.0

require (
	github.com/Marwan051/tradding_platform_game/proto/gen/go v0.0.0
	github.com/google/uuid v1.6.0
	github.com/valkey-io/valkey-glide/go/v2 v2.2.6
	google.golang.org/grpc v1.70.0
)

require (
	golang.org/x/net v0.32.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241202173237-19429a94021a // indirect
	google.golang.org/protobuf v1.36.4 // indirect
)

replace github.com/Marwan051/tradding_platform_game/proto/gen/go => ../proto/gen/go
