FROM golang:1.24-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /out/deliver .

FROM debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& useradd --create-home --home-dir /app --shell /usr/sbin/nologin appuser \
	&& mkdir -p /app /data \
	&& chown -R appuser:appuser /app /data

WORKDIR /app

COPY --from=build /out/deliver /app/deliver
COPY --from=build /src/templates /app/templates
COPY --from=build /src/static /app/static

ENV DB_PATH=/data/deliver.db

EXPOSE 8080

USER appuser

CMD ["/app/deliver"]
