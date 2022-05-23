FROM haih/xdp:latest AS xdp

FROM golang:alpine

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY *.go ./
COPY install.sh ./
RUN go build -o /setup_bash
COPY bashcni-bin/go.mod ./
COPY bashcni-bin/*.go ./
RUN go build -o /bash-cni
RUN apk add --no-cache curl jq iptables
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/kubectl

COPY --from=xdp /root/bin/bpftool ./
COPY --from=xdp /root/bin/xdp-loader ./
COPY --from=xdp /root/bin/xdp_stats ./
COPY --from=xdp /root/bin/xdp_kern.o ./

CMD [ "/setup_bash" ]
