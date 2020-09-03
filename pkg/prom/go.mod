module github.com/rancher/prometheus-auth/pkg/prom

go 1.13

require (
	github.com/grpc-ecosystem/grpc-gateway v1.14.7 // indirect
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20200706033705-4c23f9c453cd // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/rancher/prometheus-auth/pkg/data v0.0.0-20200903041626-28cfa6744693
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	google.golang.org/genproto v0.0.0-20200903010400-9bfcb5116336 // indirect
	google.golang.org/grpc v1.31.1 // indirect
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
)

replace (
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20200626085723-c448ada63d83
	github.com/rancher/prometheus-auth/pkg/data => github.com/aiwantaozi/prometheus-auth/pkg/data v0.0.0-20200903041626-28cfa6744693
)
