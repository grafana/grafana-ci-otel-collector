build:
	ocb --config config/builder-config.yml

run: 
	./grafana-collector/grafana-ci-otelcol --config config.yaml