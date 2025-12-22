BINARY := server_binary

G0_SRC := $(shell find go -name "*.go")
run: $(BINARY)
	docker compose -f 'docker-compose.yml' up -d --build --force-recreate
	
$(BINARY): go/dist $(G0_SRC) $(wildcard go/template/*.*)
	cd go && go mod tidy && go build -o ../$(BINARY) .

# make 不支持递归 ** 通配符，必须注明文件。此外，依赖目录是无效的，目录的 mtime 取决于条目的结构，而无关内部文件的元信息。
go/dist: $(wildcard solid-project/src/*) $(wildcard solid-project/public/*)
	cd solid-project && npm install && npm run build


clean:
	rm -f $(BINARY)
	rm -rf go/dist
.PHONY: clean 
