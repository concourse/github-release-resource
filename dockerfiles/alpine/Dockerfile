ARG base_image=alpine:latest
ARG builder_image=concourse/golang-builder

FROM ${builder_image} AS builder
COPY . $GOPATH/src/github.com/concourse/github-release-resource
WORKDIR $GOPATH/src/github.com/concourse/github-release-resource
ENV CGO_ENABLED 0
RUN go mod vendor
RUN go build -o /assets/out github.com/concourse/github-release-resource/cmd/out
RUN go build -o /assets/in github.com/concourse/github-release-resource/cmd/in
RUN go build -o /assets/check github.com/concourse/github-release-resource/cmd/check
RUN set -e; for pkg in $(go list ./...); do \
		go test -o "/tests/$(basename $pkg).test" -c $pkg; \
	done

FROM ${base_image} AS resource
RUN apk update && apk upgrade
RUN apk add --update bash tzdata ca-certificates
COPY --from=builder /assets /opt/resource

FROM resource AS tests
COPY --from=builder /tests /tests
RUN set -e; for test in /tests/*.test; do \
		$test; \
	done

FROM resource
