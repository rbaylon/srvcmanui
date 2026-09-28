# go.mod requires go 1.25+ (pulled in by github.com/caddyserver/certmagic) -
# keep this in sync with the `go` directive in go.mod.
FROM golang:1.25-bookworm AS base

# Move to working directory /build
WORKDIR /build

# Copy the go.mod and go.sum files to the /build directory
COPY go.mod go.sum ./

# add to fetch private repo
ADD id_rsa /root/.ssh/id_rsa
RUN chmod 700 /root/.ssh/id_rsa
RUN echo "Host github.com\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config
ARG GOPRIVATE=github.com/rbaylon/srvcman
RUN git config --global url.ssh://git@github.com/.insteadOf https://github.com/

# Install dependencies
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the application
RUN go build -o srvcmanui
RUN rm -rf /root/.ssh
# Document the port that may need to be published
EXPOSE 8000

# Start the application
CMD ["/build/srvcmanui"]

