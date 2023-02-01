FROM golang:alpine

# Install Doppler CLI
RUN wget -q -t3 'https://packages.doppler.com/public/cli/rsa.8004D9FF50437357.key' -O /etc/apk/keys/cli@doppler-8004D9FF50437357.rsa.pub && \
    echo 'https://packages.doppler.com/public/cli/alpine/any-version/main' | tee -a /etc/apk/repositories && \
    apk add doppler

RUN apk update && apk add --no-cache git
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o binary

ENTRYPOINT ["doppler", "run", "--", "/app/binary"]