FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS build
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /flowgate ./cmd/flowgate

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /flowgate /flowgate
COPY configs/flowgate.yaml /configs/flowgate.yaml
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD ["/flowgate", "healthcheck"]
ENTRYPOINT ["/flowgate", "serve", "--config", "/configs/flowgate.yaml"]
