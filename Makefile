
BINARY := server

run: $(BINARY)
	./$(BINARY)

$(BINARY): go/cmd/server/dist $(wildcard go/*)
	cd go && go build -o ../$(BINARY) ./cmd/server

# make 不支持递归 ** 通配符，必须注明文件。此外，依赖目录是无效的，目录的 mtime 取决于条目的结构，而无关内部文件的元信息。
go/cmd/server/dist: $(wildcard solid-project/src/*) $(wildcard solid-project/public/*)
	cd solid-project && npm run build


clean:
	rm -f $(BINARY)
	rm -rf go/cmd/server/dist
.PHONY: clean 

check:
	lsof -i :8080
