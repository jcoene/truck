FROM gliderlabs/alpine:3.2

ENTRYPOINT ["truck"]

EXPOSE 5000/udp

RUN apk-install ca-certificates

ADD truck_linux_amd64 /usr/bin/truck
