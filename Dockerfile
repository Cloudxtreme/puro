FROM google/golang

RUN go get github.com/tools/godep

RUN mkdir -p /gopath/src/github.com/lavab/puro
ADD . /gopath/src/github.com/lavab/puro
RUN cd /gopath/src/github.com/lavab/puro && godep go install

CMD []
ENTRYPOINT ["/gopath/bin/puro"]
