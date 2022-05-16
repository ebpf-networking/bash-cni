FROM golang:alpine

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY *.go ./
COPY bash-cni ./
COPY host-local ./
COPY apiserver ./
RUN go build -o /setup_bash
RUN apk add --no-cache curl jq iptables
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/kubectl
CMD [ "/setup_bash" ]
