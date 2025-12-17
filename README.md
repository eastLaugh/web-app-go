CICD部署

Makefile构建单文件，SCP上传。需手动启动其他服务


方法2

先运行 make 构建单文件二进制

然后允许 docker compose up -d

架构：
前后端分离，前端使用 solidjs 开发，后端使用 go 开发。

## Goods 

[添加 Good](solid-project/src/App.tsx#L18)