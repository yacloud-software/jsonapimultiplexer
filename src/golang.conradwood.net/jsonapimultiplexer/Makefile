.PHONY:	client server
server:
	cd server && go install ${LDFLAGS} jsonapimultiplexer-server.go `ls -1 *.go|grep -v jsonapimultiplexer-server.go`
client:
	cd client && go install ${LDFLAGS} jsonapimultiplexer-client.go

all:	server client
