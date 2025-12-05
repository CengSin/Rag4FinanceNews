## 开发记录

目前使用到的技术、框架、中间件

技术：

- Golang

框架：

- echo web framework
- gorm
- cleanenv
- mcp-go
- go-openai
- minIO-go

中间件：

- Temporal
- minIO

数据库：

- qdrant
- mysql

## Dependence Container

MinIO

```shell
# 启动 MinIO，控制台端口 9001，API 端口 9000
docker run -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=admin" \
  -e "MINIO_ROOT_PASSWORD=password" \
  quay.io/minio/minio server /data --console-address ":9001"
```

Qdrant

```shell
docker run -p 6333:6333 -p 6334:6334 \                                                                                    ─╯
    -v $(pwd)/qdrant_storage:/qdrant/storage \
    qdrant/qdrant
```