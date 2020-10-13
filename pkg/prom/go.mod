module github.com/rancher/prometheus-auth/pkg/prom

go 1.13

require (
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20200923013621-75df6121fbb0 // indirect
	github.com/prometheus/prometheus v2.18.2+incompatible
	github.com/rancher/prometheus-auth/pkg/data v0.0.0
)

replace (
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.0.0-20200626085723-c448ada63d83
	github.com/rancher/prometheus-auth/pkg/data => ../data
)
