FROM golang:1.24.6-alpine3.21 AS builder
RUN mkdir -p /go/src/github.com/mendersoftware/integration-test-runner
WORKDIR /go/src/github.com/mendersoftware/integration-test-runner
ADD ./ .
RUN CGO_ENABLED=0 go build

FROM golang:1.24.6-alpine3.21
EXPOSE 8080
RUN apk add git openssh python3 py3-pip gpg gpg-agent
RUN pip3 install --upgrade pyyaml PyGithub --break-system-packages
RUN mkdir -p /root/.ssh
RUN git clone https://github.com/mendersoftware/integration.git /integration
ENV PATH="/integration/extra:${PATH}"
ENV GIN_RELEASE=release
COPY --from=builder /go/src/github.com/mendersoftware/integration-test-runner/integration-test-runner /
ADD ./entrypoint /
ENTRYPOINT ["/entrypoint"]
