
BINARY := server

run: $(BINARY)
	./$(BINARY)

$(BINARY): go/cmd/server/dist $(wildcard go/*)
	cd go && go build -o ../$(BINARY) ./cmd/server

go/cmd/server/dist: $(wildcard solid-project/src/*)
	cd solid-project && npm run build


clean:
	rm -f $(BINARY)
	rm -rf go/cmd/server/dist
.PHONY: clean 

check:
	lsof -i :8080
