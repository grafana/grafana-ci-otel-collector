include .bingo/Variables.mk

metadata: $(MDATAGEN)
	$(MDATAGEN) ./pkg/dronereceiver/metadata.yaml

build: $(BINGO) $(BUILDER)
	$(BUILDER) --config config/builder-config.yml

run: 
	./collector/grafana-ci-otelcol --config config.yaml

docker-build:
	@echo "building docker container grafana-ci-otel-collector"
	docker build -e NETWORK_HOST=host.docker.internal -t grafana-ci-otel-collector .

docker-run:
	@echo "running docker container"
	docker run -it -v $$PWD:/tmp -e NETWORK_HOST=host.docker.internal -p 3333:3333 \
 		test-collector:rw-no-agent --config /tmp/config.yaml

docker: docker-build docker-run
