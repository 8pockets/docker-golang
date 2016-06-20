FROM golang:1.6

WORKDIR /go/src/app/

#RUN go get -u gopkg.in/godo.v2/cmd/godo && \
RUN go get github.com/PuerkitoBio/goquery && \
go get github.com/patrickmn/go-cache && \
go get goji.io && \
go get golang.org/x/net/context

#RUN go get github.com/Masterminds/glide && \
#export GO15VENDOREXPERIMENT=1 && \
#/go/bin/glide up

EXPOSE 5000

#CMD ["/go/bin/godo", "server", "--watch"]
