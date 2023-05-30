build:
	ocb --config config/builder-config.yml

run: 
	./collector/grafana-ci-otelcol --config config.yaml