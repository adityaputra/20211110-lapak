FROM golang:1.13-alpine AS build
WORKDIR /go/src/github.com/org/repo
COPY . .

RUN GOOS=linux GOARCH=amd64 go build -o contohlapak ./app/main.go
#RUN GOOS=linux GOARCH=amd64 go build -o contohlapak .

FROM alpine:3.12
EXPOSE 8000
COPY --from=build /go/src/github.com/org/repo/contohlapak /server/contohlapak
CMD ["/server/contohlapak"]
