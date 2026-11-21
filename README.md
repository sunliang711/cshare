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

- Go 1.26+
- Node.js 20+（构建内嵌 React Web UI）
- Redis（可选，默认使用内存存储 `memory`）

### 1. 启动服务端

```bash
cd crossshare-server
make dev
```

服务默认监听 `http://localhost:10431`：
- 客户端 Web UI：`http://localhost:10431/`
- 管理控制台：`http://localhost:10431/admin`（首次访问引导创建管理员；未初始化前客户端可正常使用）

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

Go / Gin / Uber Fx / SQLite (`modernc`) / Redis / Viper / Zerolog / JWT / React + Vite + Tailwind

### 构建与运行

```bash
cd crossshare-server

make web        # 构建前端静态资源到 ./web/dist
make dev        # 构建前端并运行服务端
make build      # 编译到 ./build/crossshare-server
make run        # 编译并运行
make test       # 运行测试 (-race)
make lint       # 代码检查
```

### Docker

```bash
cd crossshare-server
docker build -t crossshare-server .
docker run -p 10431:10431 -v $(pwd)/data:/data crossshare-server
```

CI 推送 `server/v*` 标签时自动构建多架构镜像（linux/amd64, linux/arm64）并发布到 Docker Hub。

### 配置

`config.yaml` 与环境变量（前缀 `CS_`，层级用 `_` 分隔）仅保留服务部署相关参数：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `server.port` | `10431` | 监听端口 |
| `server.tls_enable` | `false` | 是否启用 TLS |
| `server.data_dir` | `./data` | SQLite 数据库与加密主密钥 `master.key` 存放目录 |
| `server.master_key` | 空 | 可选 32 字节主密钥（64 位十六进制或 Base64）；留空则自动生成 |
| `storage.type` | `memory` | 分享内容存储后端：`memory` \| `redis` |
| `redis.addr` | `localhost:6379` | Redis 地址 |

环境变量示例：`CS_SERVER_PORT=8080`、`CS_SERVER_DATA_DIR=/data`、`CS_REDIS_ADDR=redis:6379`。

> 业务限额（TTL、文本/文件大小上限、文件数上限）、IP 限流、可选全局客户端 JWT 认证与审计策略均存储在 SQLite 中，请在 `/admin` 管理端 **「系统设置」** 页面实时配置并热生效。

### 管理端特性（`/admin`）

- **首次引导与多管理员**：首次访问 `/admin` 创建管理员（未初始化前不影响客户端正常推送与拉取）；支持登录、改密与多管理员管理。
- **运行仪表盘**：实时查看活跃分享数、占用容量、今日推送/拉取/删除次数、P2P 会话数、存储健康度与 24 小时趋势快照。
- **定时任务调度**：基于 `robfig/cron/v3`，内置清理过期审计日志、清理旧任务记录、存储健康巡检、统计快照，以及**每周自动备份到 SafeKeep**（支持测试连接与立即运行）。
- **审计日志**：记录管理员变更与客户端推送/拉取/删除/P2P 操作，支持多维筛选、详情查看及 CSV / JSON 导出。
- **环境迁移**：支持导出/预检/导入 `.csbundle` 迁移包（可选口令加密；不含客户端分享数据），新环境自动用目标主密钥重加密并热重载。

### API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/health` | 健康检查 |
| POST | `/api/v1/push/text` | 推送文本（JSON） |
| POST | `/api/v1/push/files` | 推送一个或多个文件（multipart） |
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

# 推送多个文件
share push -f ./a.txt -f ./b.jpg

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
