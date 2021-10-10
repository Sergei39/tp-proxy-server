FROM golang:1.16 AS builder
WORKDIR /build
COPY . .
RUN go build cmd/main.go
FROM ubuntu:20.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install postgresql-12 -y
USER postgres
COPY ./tables.sql .
RUN service postgresql start && \
        psql -c "CREATE USER seshishkin WITH superuser login password 'postgres';" && \
        psql -c "ALTER ROLE seshishkin WITH PASSWORD 'postgres';" && \
        createdb -O seshishkin proxy && \
        psql -d proxy < ./tables.sql && \
        service postgresql stop

#VOLUME ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

USER root

WORKDIR /proxy
COPY --from=builder /build/main .

COPY . .

EXPOSE 8088
EXPOSE 8000
EXPOSE 5432

CMD service postgresql start && ./main