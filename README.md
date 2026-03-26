# CrossShare

跨设备内容分享平台 —— 在任意设备之间快速分享文本和文件。

设备 A 推送内容，获取一个短 key；设备 B 凭 key 拉取，支持自动过期回收。

## 项目结构

```
.
├── crossshare-server/   # 服务端（HTTP API + 内嵌 Web UI）
├── crossshare-cli/      # 命令行客户端 (share)
└── .github/workflows/   # CI/CD（Docker 镜像 & GitHub Release）
```

## 快速开始

### 前置依赖

- Go 1.25+
- Redis

### 1. 启动服务端

```bash
cd crossshare-server
make dev
```

服务默认监听 `http://localhost:10431`，内嵌 Web UI 可直接通过浏览器访问。

### 2. 使用 CLI

```bash
cd crossshare-cli
make build

# 推送文本
./build/share push "hello world"
# => Key: A8k2dP

# 拉取内容
./build/share pull A8k2dP
```

---

## crossshare-server

HTTP API 服务端，提供内容的上传、拉取、删除和自动过期功能，内嵌轻量 Web UI。

### 技术栈

Go / Gin / Uber Fx / Redis / Viper / Zerolog / JWT

### 构建与运行

```bash
cd crossshare-server

make dev        # 开发模式运行
make build      # 编译到 ./build/crossshare-server
make run        # 编译并运行
make test       # 运行测试
make lint       # 代码检查
```

### Docker

```bash
cd crossshare-server
docker build -t crossshare-server .
docker run -p 10431:10431 crossshare-server
```

CI 推送 `server/v*` 标签时自动构建多架构镜像（linux/amd64, linux/arm64）并发布到 Docker Hub。

### 配置

通过 `config.yaml` 或环境变量（前缀 `CS_`，层级用 `_` 分隔）配置：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `server.port` | `10431` | 监听端口 |
| `server.tls_enable` | `false` | 是否启用 TLS |
| `auth.enable` | `false` | 是否启用 JWT 认证 |
| `auth.jwt_secret` | `change-me-in-production` | JWT 密钥 |
| `business.default_ttl` | `600` | 默认过期时间（秒） |
| `business.max_ttl` | `2592000` | 最大 TTL（30 天） |
| `business.text_json_limit` | `1048576` | 文本大小上限（1 MB） |
| `business.binary_push_limit` | `20971520` | 文件大小上限（20 MB） |
| `ratelimit.enable` | `true` | 是否启用限流 |
| `ratelimit.requests_per_minute` | `60` | 每 IP 每分钟请求数 |
| `redis.addr` | `localhost:6379` | Redis 地址 |

环境变量示例：`CS_SERVER_PORT=8080`、`CS_REDIS_ADDR=redis:6379`。

### API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/health` | 健康检查 |
| POST | `/api/v1/push/text` | 推送文本（JSON） |
| POST | `/api/v1/push/binary` | 推送文件（流式） |
| POST | `/api/v1/push` | 统一推送（按 Content-Type 自动分发） |
| GET | `/api/v1/pull/:key` | 拉取内容 |
| DELETE | `/api/v1/pull/:key` | 删除内容 |

### Systemd

```bash
sudo crossshare-server install                              # 基本安装
sudo crossshare-server install --config /path/to/config.yaml  # 使用自定义配置
sudo crossshare-server install --user crossshare --start      # 指定运行用户并立即启动
sudo crossshare-server install --uninstall                    # 卸载
```

---

## crossshare-cli

服务端的命令行客户端，二进制名称为 `share`。

### 构建与安装

```bash
cd crossshare-cli

make build      # 编译到 ./build/share
make install    # 安装到 $GOPATH/bin
```

### 全局参数

| 参数 | 环境变量 | 默认值 | 说明 |
|------|---------|--------|------|
| `-s, --server` | `CROSSSHARE_SERVER` | `http://localhost:10431` | 服务器地址 |
| `-t, --token` | `CROSSSHARE_TOKEN` | 空 | JWT 认证 token |

### 使用示例

```bash
# 健康检查
share health

# 推送文本
share push "hello world"

# 设置过期时间
share push "hello" --ttl 7200

# 从管道推送
echo "piped content" | share push -

# 推送文件
share push -f ./report.pdf

# 拉取内容
share pull A8k2dP

# 拉取并保存到指定文件
share pull A8k2dP -o output.bin

# 拉取后自动删除
share pull A8k2dP --delete

# 删除
share delete A8k2dP
```

---

## CI/CD

| 工作流 | 触发条件 | 说明 |
|--------|---------|------|
| `release-server.yml` | 推送 `server/v*` 标签 | 构建多架构 Docker 镜像并推送至 Docker Hub |
| `release-cli.yml` | 推送 `cli/v*` 标签 | 交叉编译 CLI 并创建 GitHub Release |

### 发布示例

```bash
# 发布服务端 Docker 镜像
git tag server/v1.0.0
git push origin server/v1.0.0

# 发布 CLI
git tag cli/v1.0.0
git push origin cli/v1.0.0
```

CLI 发布产物覆盖以下平台：

- `linux/amd64`、`linux/arm64`
- `darwin/amd64`、`darwin/arm64`
- `windows/amd64`

## License

MIT
