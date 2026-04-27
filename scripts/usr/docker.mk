.PHONY: docker
docker:
	docker build -f build/Dockerfile -t ghcr.io/sotbi-llc/gh-actions .
