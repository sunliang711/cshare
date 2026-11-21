variable "image" {
  type    = string
  default = "sunliang711/crossshare-server:v0.3.0"
}

job "crossshare" {
  datacenters = ["dc1"]
  type        = "service"

  group "server" {
    count = 1

    # 多节点 Nomad 集群建议解除下方注释并锁定运行节点，避免重新调度漂移到其他节点找不到本地 SQLite 数据：
    # constraint {
    #   attribute = "${attr.unique.hostname}"
    #   value     = "your-node-hostname"
    # }

    update {
      max_parallel = 1
    }

    network {
      port "http" {
        static = 8400
        to     = 10431
        host_network = "loopback"
      }
    }

    service {
      name     = "crossshare"
      provider = "nomad"
      port     = "http"

      check {
        name     = "health"
        type     = "http"
        path     = "/api/v1/health"
        interval = "10s"
        timeout  = "2s"
      }
    }

    volume "crossshare" {
      type      = "host"
      source    = "crossshare"
      read_only = false
    }

    task "crossshare" {
      driver = "docker"

      # 环境变量优先级高于容器内置 config.yaml（前缀 CS_，层级用 _ 分隔）。
      env {
        # --- 基础服务与持久化路径 ---
        CS_SERVER_PORT     = "10431"
        CS_SERVER_DATA_DIR = "/data"
        # 可选：固定 32 字节加密主密钥（64 位十六进制或 Base64）；留空则自动生成并持久化到 /data/master.key
        # CS_SERVER_MASTER_KEY = ""

        # --- 分享内容存储后端：memory | redis ---
        CS_STORAGE_TYPE = "redis"
        # 若使用外部 Redis，请将 CS_STORAGE_TYPE 改为 "redis" 并取消下方注释：
        # CS_REDIS_ADDR     = "127.0.0.1:6379"
        # CS_REDIS_USERNAME = ""
        # CS_REDIS_PASSWORD = ""
        # CS_REDIS_DB       = "0"

        # --- 可选：TLS ---
        # CS_SERVER_TLS_ENABLE = "false"
        # CS_SERVER_CRT_PATH   = ""
        # CS_SERVER_KEY_PATH   = ""

        # 注：业务限额（TTL、文本/文件大小上限、单次文件数）、IP 限流、可选全局客户端 JWT 认证
        # 与审计策略统一持久化在 SQLite 数据库（/data/crossshare.db），请在 /admin 管理端「系统设置」中配置。
      }

      volume_mount {
        volume      = "crossshare"
        destination = "/data"
        read_only   = false
      }

      config {
        image = var.image
        force_pull = false
        ports = ["http"]
        logging {
          type = "json-file"
          config {
            max-file = "2"
            max-size = "1m"
          }
        }
      }

      identity {
        name = "vault_default"
        aud  = ["vault.io"]
        file = true
        ttl  = "1h"
      }

      vault {
        cluster = "default"
        role    = "working-service"
      }

      template {

    data = <<EOH
CS_REDIS_DB       = "0"
{{ with secret "kv/data/working-service/redis" }}
{{ $redis_password := .Data.data.redis_password }}
{{ range nomadService "redis" }}
CS_REDIS_ADDR={{ .Address }}:{{ .Port }}
CS_REDIS_USERNAME=
CS_REDIS_PASSWORD={{ $redis_password }}
{{ end }}
{{ end }}
EOH
    destination = "secrets/redis.env"
    env         = true
    change_mode = "restart"
  }
      service {
          name = "crossshare"
          port = "http"
          provider = "nomad"
          # 服务中的地址从driver(docker)中取得
          address_mode = "driver"
      }

      resources {
        cpu    = 200
        memory = 256
      }
    }
  }
}
