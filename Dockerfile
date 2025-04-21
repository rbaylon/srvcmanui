# Use Go 1.23 bookworm as base image
FROM golang:1.24-bookworm AS base

# Move to working directory /build
WORKDIR /build

# Copy the go.mod and go.sum files to the /build directory
COPY go.mod go.sum ./

# add to fetch private repo
ARG GOPRIVATE=github.com/rbaylon/srvcman
RUN git config --global url.ssh://git@github.com/.insteadOf https://github.com/

# Install dependencies
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the application
RUN go build -o srvcman

# Document the port that may need to be published
EXPOSE 8000

# Start the application
CMD ["/build/srvcmanui"]

