include .bingo/Variables.mk

build: $(BINGO) $(BUILDER)
	$(BUILDER) --config config/builder-config.yml

run: 
	./collector/grafana-ci-otelcol --config config.yaml