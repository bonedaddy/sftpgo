FROM golang:alpine as builder

RUN apk add --no-cache git gcc g++ ca-certificates \
  && go get -d github.com/drakkan/sftpgo
WORKDIR /go/src/github.com/drakkan/sftpgo
ARG TAG
ARG FEATURES
# Use --build-arg TAG=LATEST for latest tag. Use e.g. --build-arg TAG=0.9.6 for a specific tag/commit. Otherwise HEAD (master) is built.
RUN git checkout $(if [ "${TAG}" = LATEST ]; then echo `git rev-list --tags --max-count=1`; elif [ -n "${TAG}" ]; then echo "${TAG}"; else echo HEAD; fi)
RUN go build -i $(if [ -n "${FEATURES}" ]; then echo "-tags ${FEATURES}"; fi) -ldflags "-s -w -X github.com/drakkan/sftpgo/utils.commit=`git describe --always --dirty` -X github.com/drakkan/sftpgo/utils.date=`date -u +%FT%TZ`" -o /go/bin/sftpgo

FROM alpine:latest

RUN apk add --no-cache ca-certificates su-exec \
  && mkdir -p /data /etc/sftpgo /srv/sftpgo/config /srv/sftpgo/web /srv/sftpgo/backups

# git and rsync are optional, uncomment the next line to add support for them if needed.
#RUN apk add --no-cache git rsync

COPY --from=builder /go/bin/sftpgo /bin/
COPY --from=builder /go/src/github.com/drakkan/sftpgo/sftpgo.json /etc/sftpgo/sftpgo.json
COPY --from=builder /go/src/github.com/drakkan/sftpgo/templates /srv/sftpgo/web/templates
COPY --from=builder /go/src/github.com/drakkan/sftpgo/static /srv/sftpgo/web/static
COPY docker-entrypoint.sh /bin/entrypoint.sh
RUN chmod +x /bin/entrypoint.sh

VOLUME [ "/data", "/srv/sftpgo/config", "/srv/sftpgo/backups" ]
EXPOSE 2022 8080

ENTRYPOINT ["/bin/entrypoint.sh"]
CMD ["serve"]
