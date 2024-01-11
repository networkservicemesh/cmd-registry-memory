FROM golang:1.20.12 as go
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOBIN=/bin
ARG BUILDARCH=amd64
RUN go install github.com/go-delve/delve/cmd/dlv@v1.22.0
RUN go install github.com/grpc-ecosystem/grpc-health-probe@v0.4.24
ADD https://github.com/spiffe/spire/releases/download/v1.8.7/spire-1.8.7-linux-${BUILDARCH}-musl.tar.gz .
RUN tar xzvf spire-1.8.7-linux-${BUILDARCH}-musl.tar.gz -C /bin --strip=2 spire-1.8.7/bin/spire-server spire-1.8.7/bin/spire-agent

FROM go as build
WORKDIR /build
COPY go.mod go.sum ./
COPY pkg ./pkg
RUN go build ./pkg/imports
COPY . .
RUN go build -o /bin/registry-memory .

FROM build as test
CMD go test -test.v ./...

FROM test as debug
CMD dlv -l :40000 --headless=true --api-version=2 test -test.v ./...

FROM alpine as runtime
COPY --from=build /bin/registry-memory /bin/registry-memory
COPY --from=build /bin/grpc-health-probe /bin/grpc-health-probe
ENTRYPOINT ["/bin/registry-memory"]