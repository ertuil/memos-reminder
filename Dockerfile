FROM docker.m.daocloud.io/golang:1.21 as builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 
    
WORKDIR /app

COPY . .

RUN go build -v -trimpath -gcflags="all=-m -l -l" -ldflags "-s -w" -o memos-reminder .

FROM docker.m.daocloud.io/ubuntu:latest
RUN sed -i "s@//.*archive.ubuntu.com@//mirrors.ustc.edu.cn@g" /etc/apt/sources.list
RUN apt-get -qq update \
    && apt-get -qq install -y --no-install-recommends ca-certificates curl
RUN update-ca-certificates
WORKDIR /app
COPY --from=builder /app/memos-reminder /app/memos-reminder
RUN chmod +x /app/memos-reminder
EXPOSE 8880

CMD ["/app/memos-reminder", "--verbose", "--config", "/app/data/config.yaml", "--database", "/app/data/database.db"]