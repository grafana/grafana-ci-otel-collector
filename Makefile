include .bingo/Variables.mk

metadata: $(MDATAGEN)
	$(MDATAGEN) ./pkg/dronereceiver/metadata.yaml
	$(MDATAGEN) ./pkg/githubactionsreceiver/metadata.yaml

build: $(BINGO) $(BUILDER)
	$(BUILDER) --config config/builder-config.yml

run: 
	./collector/grafana-ci-otelcol --config config.yaml

dev: metadata build run

docker-build:
	@echo "building docker container grafana-ci-otel-collector"
	docker build -t grafana-ci-otel-collector .

docker-run:
	@echo "running docker container"
	docker run -it -v $$PWD:/tmp -p 3333:3333  \
 		grafana-ci-otel-collector:latest --config /tmp/config.yaml

docker: docker-build docker-run

drone:
	jsonnet -J .drone/vendor/ .drone/drone.jsonnet > jsonnetfile
	drone jsonnet --stream \
		--format \
		--source jsonnetfile \
		--target .drone.yml
	rm jsonnetfile
