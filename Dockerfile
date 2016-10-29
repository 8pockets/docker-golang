FROM golang:1.7
ENV GOPATH /go

RUN curl https://glide.sh/get | sh

#glideを使うプロジェクトは$GOPATH/src/ディレクトリ以下に作成
WORKDIR /go/src/app
COPY glide.yaml /go/src/app/glide.yaml
RUN glide install -v

EXPOSE 5000

RUN go get -u gopkg.in/godo.v2/cmd/godo
CMD ["/go/bin/godo", "server", "--watch"]
