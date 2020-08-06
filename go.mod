module github.com/rancher/prometheus-auth

go 1.13

replace github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20200626085723-c448ada63d83

require (
	github.com/prometheus/prometheus v2.18.2+incompatible
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	google.golang.org/genproto v0.0.0-20200804151602-45615f50871c // indirect
	google.golang.org/grpc v1.31.0 // indirect
)
