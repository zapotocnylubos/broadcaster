FROM golang:1.20.1-alpine as build

WORKDIR /app
COPY go.mod broadcaster.go ./
RUN go build -o broadcaster


FROM golang:1.20.1-alpine

COPY --from=build /app/broadcaster /app/broadcaster

ENTRYPOINT ["/app/broadcaster"]
EXPOSE 80
