# This Dockerfile builds a Docker image which only contains the necessary
# binaries to run the program, as opposed to a full Linux+Golang install
# which can weigh close to 1GB. To keep with the principle of microservices
# that they need to be light and portable, this resulting image is <10MB in size

# This binary will run on any Docker system which is running Linux containers

# Step 1 is to use a lightweight golang alpine image to build to fetch the
# dependencies and build the binary
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
RUN CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o msa .
WORKDIR /dist
RUN cp /build/msa .

# Step 2 starts from a scratch (completely blank) image and only copies the
# necessary dependencies and the binary to run this specific binary
FROM scratch

# Copy our static executable.
COPY --from=builder /dist/msa /

# Run the hello binary.
EXPOSE 8888
ENTRYPOINT ["./msa"]
