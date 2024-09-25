module github.com/grafana/grafana-ci-otel-collector

go 1.23.1

replace github.com/grafana/grafana-ci-otel-collector/receiver/dronereceiver => ./receiver/dronereceiver

replace github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver => ./receiver/githubactionsreceiver

replace github.com/grafana/grafana-ci-otel-collector/internal/traceutils => ./internal/traceutils

replace github.com/grafana/grafana-ci-otel-collector/internal/semconv => ./internal/semconv

replace github.com/grafana/grafana-ci-otel-collector/internal/sharedcomponent => ./internal/sharedcomponent

require github.com/grafana/grafana-ci-otel-collector/receiver/dronereceiver v0.0.0-00010101000000-000000000000

require github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver v0.0.0-00010101000000-000000000000

require (
	github.com/99designs/httpsignatures-go v0.0.0-20170731043157-88528bf4ca7e // indirect
	github.com/bradleyfalzon/ghinstallation/v2 v2.11.0 // indirect
	github.com/drone/drone-go v1.7.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-github/v62 v62.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grafana/grafana-ci-otel-collector/internal/semconv v0.0.0-00010101000000-000000000000 // indirect
	github.com/grafana/grafana-ci-otel-collector/internal/sharedcomponent v0.0.0-00010101000000-000000000000 // indirect
	github.com/grafana/grafana-ci-otel-collector/internal/traceutils v0.0.0-00010101000000-000000000000 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/rs/cors v1.11.1 // indirect
	go.opentelemetry.io/collector v0.109.0 // indirect
	go.opentelemetry.io/collector/client v1.15.0 // indirect
	go.opentelemetry.io/collector/component v0.109.0 // indirect
	go.opentelemetry.io/collector/config/configauth v0.109.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.15.0 // indirect
	go.opentelemetry.io/collector/config/confighttp v0.109.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.15.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.109.0 // indirect
	go.opentelemetry.io/collector/config/configtls v1.15.0 // indirect
	go.opentelemetry.io/collector/config/internal v0.109.0 // indirect
	go.opentelemetry.io/collector/confmap v1.15.0 // indirect
	go.opentelemetry.io/collector/consumer v0.109.0 // indirect
	go.opentelemetry.io/collector/consumer/consumerprofiles v0.109.0 // indirect
	go.opentelemetry.io/collector/extension v0.109.0 // indirect
	go.opentelemetry.io/collector/extension/auth v0.109.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.15.0 // indirect
	go.opentelemetry.io/collector/pdata v1.15.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.109.0 // indirect
	go.opentelemetry.io/collector/receiver v0.109.0 // indirect
	go.opentelemetry.io/collector/semconv v0.109.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240822170219-fc7c04adadcd // indirect
	google.golang.org/grpc v1.66.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
