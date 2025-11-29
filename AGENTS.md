永远做最小的修改
听从用户指示，宁可少做，不要多做
最小修改原则
非必要不修改
优先修改前端，尽量不要动后端

新API流程：
修改api/文件下的yaml文件，确保yaml文件符合openAPI规范

使用oapi-codegen工具生成代码，见 go/api/gen.go 文件

根据情况少量生成 .http 文件

修改.env.example文件