module github.com/scttfrdmn/bucktooth

go 1.24.0

replace github.com/scttfrdmn/agenkit/agenkit-go => ../../agenkit/agenkit-go

require (
	github.com/bwmarrin/discordgo v0.28.1
	github.com/gorilla/websocket v1.5.3
	github.com/prometheus/client_golang v1.20.5
	github.com/rs/zerolog v1.32.0
	github.com/scttfrdmn/agenkit/agenkit-go v0.0.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/prometheus v0.54.0
	go.opentelemetry.io/otel/trace v1.38.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
)
