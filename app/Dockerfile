FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/app .


FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/app /app/app

EXPOSE 8280

ENTRYPOINT ["/app/app"]

