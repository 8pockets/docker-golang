FROM golang:onbuild

RUN go get github.com/PuerkitoBio/goquery \
go get github.com/patrickmn/go-cache

EXPOSE 5000
