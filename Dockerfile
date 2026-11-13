FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/build ./web/build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /ratatosk .
# The sites volume inherits this ownership on first mount
RUN mkdir -p /sites

FROM gcr.io/distroless/static:nonroot
COPY --from=build /ratatosk /ratatosk
COPY --from=build --chown=nonroot:nonroot /sites /var/www/html
ENV ADDR=:8000 SITES_DIR=/var/www/html
EXPOSE 8000
ENTRYPOINT ["/ratatosk"]
