############################
# STEP 1 build executable binary
############################
FROM golang:1.13.8-alpine AS builder
# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git
WORKDIR /build
COPY . .
# Fetch dependencies.
# Using go get.
RUN go get -d -v
# Build the binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o bluebook .
WORKDIR /dist
RUN cp /build/bluebook .
############################
# STEP 2 build a small image
############################
FROM scratch
# Copy our static executable.
COPY --from=builder /dist/bluebook /
# RUN ls
# Run the hello binary.
EXPOSE 8888
ENTRYPOINT ["./bluebook"]
