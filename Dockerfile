# Build with the golang image
FROM golang:1.15-alpine AS build

# Add git
RUN apk add git

# Set workdir
WORKDIR /work

# Add dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build
COPY . .
ARG VERSION=development
RUN CGO_ENABLED=0 go build -ldflags "-X github.com/StatCan/ingress-istio-controller/pkg/controller.controllerAgentVersion=${VERSION}"

# Generate final image
FROM scratch
COPY --from=build /work/ingress-istio-controller /ingress-istio-controller
ENTRYPOINT [ "/ingress-istio-controller" ]
