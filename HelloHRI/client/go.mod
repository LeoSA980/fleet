module HelloHRI/client

go 1.24.3

require (
	HelloHRI/proto v0.0.0
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
)

replace HelloHRI/proto => ../proto
