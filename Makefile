GOPATH  	= $(CURDIR)
AGENT   	= $(GOPATH)/cmd/agent
COORDINATOR = $(GOPATH)/cmd/coordinator

all:
	@go get github.com/google/gopacket
	@go get github.com/codahale/hdrhistogram
	@go get -u github.com/gorilla/mux
	@go get google.golang.org/grpc
	@go get gopkg.in/yaml.v2
	@go get github.com/mattn/go-sqlite3
	@cd $(AGENT) && go build -ldflags -s
	@cd $(COORDINATOR) && go build -ldflags -s
	@rm -rf bin && mkdir bin
	@mv $(AGENT)/agent bin/agent
	@mv $(COORDINATOR)/coordinator bin/coordinator
