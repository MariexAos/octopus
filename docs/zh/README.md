<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Gin-1.11+-008EC4?style=for-the-badge&logo=gin" alt="Gin Version">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/coverage-90%25+-brightgreen?style=for-the-badge" alt="Coverage">
</p>

<h1 align="center">Octopus - 短链接服务</h1>

<p align="center">
  <b>基于 Go 最佳实践构建的高性能、可扩展短链接服务</b>
</p>

<p align="center">
  <a href="#特性">特性</a> •
  <a href="#快速开始">快速开始</a> •
  <a href="#api-文档">API</a> •
  <a href="#配置">配置</a> •
  <a href="#部署">部署</a> •
  <a href="#架构">架构</a>
</p>

---

## 特性

- **高性能** - 基于 Gin 框架，配合 Redis 缓存
- **冲突检测** - 使用 Redis 布隆过滤器快速检测重复
- **灵活编码** - Base32 编码，支持 4-6 位短码
- **302 重定向** - 标准 HTTP 重定向，支持查询参数合并
- **数据分析** - 实时 PV/UV 统计和来源分析
- **异步处理** - 使用 RocketMQ 处理高吞吐访问日志
- **生产就绪** - 支持 Docker、Kubernetes 和 docker-compose
- **测试完善** - 单元测试覆盖率 90%+

## 快速开始

### 环境要求

- Go 1.26+
- Redis 7.0+
- MySQL 8.0+
- （可选）RocketMQ 5.0+

### 使用 Docker Compose 运行

```bash
# 克隆仓库
git clone https://github.com/MariexAos/octopus.git
cd octopus

# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps
```

### 本地运行

```bash
# 安装依赖
go mod download

# 启动基础设施（Redis、MySQL）
docker-compose up -d redis mysql

# 运行服务
make run

# 或使用热重载
make dev
```

服务将在 `http://localhost:8080` 启动

### API 使用示例

**生成短链接**

```bash
curl -X POST http://localhost:8080/api/v1/shortlink/generate \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/very/long/url",
    "params": {
      "utm_source": "newsletter",
      "campaign": "spring2024"
    }
  }'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "short_link": "http://localhost:8080/AbCd",
    "short_code": "AbCd",
    "original_url": "https://example.com/very/long/url"
  }
}
```

**访问短链接**

```bash
curl -I http://localhost:8080/AbCd

# 响应: HTTP/1.1 302 Found
# Location: https://example.com/very/long/url?utm_source=newsletter&campaign=spring2024
```

**获取统计数据**

```bash
curl http://localhost:8080/api/v1/analytics/AbCd
```

响应：
```json
{
  "code": 0,
  "data": {
    "short_code": "AbCd",
    "pv": 10000,
    "uv": 3500,
    "top_sources": [
      {"source": "google.com", "count": 4000},
      {"source": "twitter.com", "count": 2500}
    ]
  }
}
```

## API 文档

| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/api/v1/shortlink/generate` | 生成新的短链接 |
| GET | `/{shortCode}` | 重定向到原始 URL |
| GET | `/api/v1/analytics/{shortCode}` | 获取统计数据 |
| GET | `/swagger/index.html` | Swagger UI |

Swagger UI 地址：`http://localhost:8080/swagger/index.html`

## 配置

配置文件：`configs/config.yaml`

```yaml
server:
  port: 8080
  mode: release  # debug, release, test

database:
  mysql:
    dsn: "user:password@tcp(localhost:3306)/shortlink?charset=utf8mb4&parseTime=True"
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0

bloom:
  capacity: 1000000000  # 10 亿
  error_rate: 0.01      # 1%

rocketmq:
  nameserver: "localhost:9876"
  topic: "access_log"
```

### 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `SERVER_PORT` | 服务端口 | `8080` |
| `MYSQL_DSN` | MySQL 连接字符串 | - |
| `REDIS_ADDR` | Redis 地址 | `localhost:6379` |
| `ROCKETMQ_NAMESERVER` | RocketMQ Name Server | - |

## 架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   客户端    │────▶│   Gin API   │────▶│   Service   │
└─────────────┘     └─────────────┘     └─────────────┘
                           │                    │
                           ▼                    ▼
                    ┌─────────────┐     ┌─────────────┐
                    │  Middleware │     │  Repository │
                    └─────────────┘     └─────────────┘
                                               │
                              ┌────────────────┼────────────────┐
                              ▼                ▼                ▼
                       ┌───────────┐    ┌───────────┐    ┌───────────┐
                       │   Redis   │    │   MySQL   │    │ RocketMQ  │
                       └───────────┘    └───────────┘    └───────────┘
```

### 项目结构

```
octopus/
├── cmd/
│   └── server/          # 应用入口
├── internal/
│   ├── config/          # 配置管理
│   ├── encoder/         # Base32 编码器
│   ├── handler/         # HTTP 处理器
│   ├── model/           # 数据模型
│   ├── mq/              # RocketMQ 生产者/消费者
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   └── mocks/           # 测试用 Mock 实现
├── pkg/
│   ├── middleware/      # HTTP 中间件
│   └── util/            # 工具函数
├── configs/             # 配置文件
├── deployments/         # Docker & K8s 配置
├── scripts/             # 数据库迁移脚本
├── docs/                # 文档
├── Makefile             # 构建自动化
└── docker-compose.yaml  # 本地开发环境
```

## 部署

### Docker

```bash
# 构建镜像
docker build -f deployments/docker/Dockerfile -t octopus:latest .

# 运行容器
docker run -d \
  --name octopus \
  -p 8080:8080 \
  -e MYSQL_DSN="user:pass@tcp(mysql:3306)/shortlink" \
  -e REDIS_ADDR="redis:6379" \
  octopus:latest
```

### Kubernetes

```bash
# 应用配置
kubectl apply -f deployments/k8s/
```

## 开发

### Makefile 命令

```bash
make help          # 显示所有可用命令
make build         # 构建二进制文件
make run           # 运行服务
make test          # 运行所有测试
make test-coverage # 运行测试并生成覆盖率报告
make lint          # 运行代码检查
make docker-build  # 构建 Docker 镜像
make swagger       # 生成 Swagger 文档
make migrate-up    # 执行数据库迁移
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行测试并生成覆盖率
make test-coverage

# 查看覆盖率报告
open coverage.html
```

## 路线图

- [ ] 自定义短码支持
- [ ] 链接过期管理
- [ ] 二维码生成
- [ ] 批量导入/导出
- [ ] 管理后台
- [ ] 限流控制
- [ ] GraphQL API

## 贡献

欢迎贡献代码！提交 PR 前请阅读贡献指南。

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 致谢

- [Gin](https://github.com/gin-gonic/gin) - Web 框架
- [GORM](https://gorm.io/) - ORM
- [go-redis](https://github.com/redis/go-redis) - Redis 客户端
- [RocketMQ](https://rocketmq.apache.org/) - 消息队列

---

<p align="center">
  由 <a href="https://github.com/MariexAos">MariexAos</a> 用心打造
</p>
