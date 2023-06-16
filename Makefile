include .bingo/Variables.mk

metadata: $(MDATAGEN)
	$(MDATAGEN) ./pkg/dronereceiver/metadata.yaml

build: $(BINGO) $(BUILDER)
	$(BUILDER) --config config/builder-config.yml

run: 
	./collector/grafana-ci-otelcol --config config.yaml