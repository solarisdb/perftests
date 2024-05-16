FROM golang:1.22 as builder
LABEL stage=builder

WORKDIR /usr/src
COPY .. .

RUN apt update && apt -y --no-install-recommends install openssh-client git && \
    mkdir -p -m 0700 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts && \
    git config --global url."git@github.com:".insteadOf "https://github.com"

RUN --mount=type=ssh CGO_ENABLED=0 make all

FROM alpine:3.16.0

ADD https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.12/grpc_health_probe-linux-amd64 /bin/grpc_health_probe

RUN chmod +x /bin/grpc_health_probe

WORKDIR /app

COPY --from=builder /usr/src/build/perftests .
COPY --from=builder /usr/src/test-scripts/* ./test-scripts/

CMD exec /app/perftests start $PERFTEST_CFG_FILES
