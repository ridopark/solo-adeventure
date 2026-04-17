FROM golang:1.25-bookworm AS builder

WORKDIR /build

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /solo-adeventure-server ./cmd/solo-adeventure-server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /solo-adeventure-server /solo-adeventure-server

EXPOSE 8084

ENTRYPOINT ["/solo-adeventure-server"]
