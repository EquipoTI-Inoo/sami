FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /coachia-backend .

FROM alpine:3.21
RUN apk add --no-cache wget
WORKDIR /app
COPY --from=build /coachia-backend ./coachia-backend
EXPOSE 3000
ENTRYPOINT ["./coachia-backend"]
