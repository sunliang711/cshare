# CrossShare CLI

crossshare-server 的命令行客户端，用于跨设备分享文本和文件。

## 构建

```bash
make build    # 生成 ./share 二进制
make install  # 安装到 $GOPATH/bin
```

## 全局参数

| 参数 | 环境变量 | 默认值 | 说明 |
|------|---------|--------|------|
| `-s, --server` | `CROSSSHARE_SERVER` | `http://localhost:10431` | 服务器地址 |
| `-t, --token` | `CROSSSHARE_TOKEN` | 空 | JWT 认证 token |

## 子命令

### health — 健康检查

```bash
share health
```

### push — 推送文本或文件

```bash
# 推送文本
share push "hello world"

# 设置过期时间（秒）
share push "hello" --ttl 7200

# 从 stdin 管道推送
echo "piped content" | share push -

# 推送文件
share push -f ./report.pdf

# 推送文件并指定文件名和 TTL
share push -f ./notes.txt --filename custom.txt --ttl 86400

# 指定 content-type
share push -f ./data.json --content-type application/json
```

| 参数 | 说明 |
|------|------|
| `[text]` | 要推送的文本内容，`-` 表示从 stdin 读取 |
| `-f, --file` | 上传文件路径 |
| `--ttl` | 过期时间（秒），默认由服务端决定 |
| `--filename` | 自定义文件名 |
| `--content-type` | 自定义 MIME 类型 |

### pull — 拉取内容

```bash
# 拉取（文本打印到 stdout，二进制按原文件名保存）
share pull A8k2dP

# 保存到指定文件
share pull A8k2dP -o output.bin

# 以 JSON 格式返回（仅文本内容）
share pull A8k2dP --json

# 拉取后自动删除
share pull A8k2dP --delete
```

| 参数 | 说明 |
|------|------|
| `<key>` | 分享 key（必填） |
| `-o, --output` | 保存到指定文件路径 |
| `--json` | 强制以 JSON 格式返回 |
| `--delete` | 拉取成功后删除 |

### delete — 删除内容

```bash
share delete A8k2dP
```

## 接口对照

| 子命令 | HTTP 方法 | API 路径 |
|--------|----------|----------|
| `health` | GET | `/api/v1/health` |
| `push` (文本) | POST | `/api/v1/push/text` |
| `push -f` (文件) | POST | `/api/v1/push/binary` |
| `pull` | GET | `/api/v1/pull/:key` |
| `delete` | DELETE | `/api/v1/pull/:key` |
