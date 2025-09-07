module github.com/ceyewan/gochat/im-task

go 1.21

require (
	github.com/ceyewan/gochat/api/gen/im_repo/v1 v0.0.0
	github.com/ceyewan/gochat/api/gen/im_task/v1 v0.0.0
	github.com/ceyewan/gochat/internal/kafka v0.0.0
	github.com/ceyewan/gochat/internal/pkg v0.0.0
	github.com/ceyewan/gochat/internal/proto v0.0.0
	github.com/ceyewan/gochat/internal/types v0.0.0
	github.com/ceyewan/gochat/pkg/log v0.0.0
	github.com/ceyewan/gochat/pkg/middleware v0.0.0
	github.com/ceyewan/gochat/pkg/redis v0.0.0
	github.com/ceyewan/gochat/pkg/tracing v0.0.0
	github.com/ceyewan/gochat/pkg/wrappers v0.0.0
	github.com/google/uuid v1.6.0
	github.com/spf13/viper v1.18.2
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/segmentio/kafka-go v0.4.45 // indirect
	github.com/twmb/franz-go v1.16.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0 // indirect
	go.opentelemetry.io/otel/sdk v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231212172506-995d672761c0 // indirect
)
