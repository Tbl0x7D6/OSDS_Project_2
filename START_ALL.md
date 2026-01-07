# 🚀 Bitcoin Visualizer 启动指南

## 系统架构

```
Frontend (React) :5173
    ↓
API Server (Express) :3000
    ↓
CLI Client (Go)
    ↓
Miner Node (Go) :8001
```

## 快速启动（4个步骤）

### 1. 编译 Go 客户端
```bash
cd /workspaces/go
go build -o bin/client cmd/client/main.go
```

### 2. 启动矿工节点（终端1）
```bash
cd /workspaces/go
./bin/miner -id 1 -address 0.0.0.0:8001 -difficulty 6
```

### 3. 启动 API 服务器（终端2）
```bash
cd /workspaces/go/WebUI
node api-server.mjs
```

### 4. 启动前端开发服务器（终端3）
```bash
cd /workspaces/go/WebUI
pnpm dev
```

## 访问应用

打开浏览器访问: **http://localhost:5173**

## 验证服务状态

### 检查矿工节点
```bash
./bin/client blockchain -miner localhost:8001
```

### 检查 API 服务器
```bash
curl http://localhost:3000/api/health
```

### 检查前端
```bash
curl http://localhost:5173
```

## 故障排查

### Client 无法连接到 Miner
✅ **已解决** - CLI 可以正常连接到 miner

检查：
```bash
# 1. 确认 miner 正在运行
ps aux | grep miner

# 2. 确认端口监听
netstat -tlnp | grep 8001

# 3. 测试连接
./bin/client blockchain -miner localhost:8001
```

### API 服务器无法连接
检查：
```bash
# 1. 确认服务运行
curl http://localhost:3000/api/health

# 2. 测试区块链 API
curl http://localhost:3000/api/blockchain/status

# 3. 查看错误日志
# 在运行 node api-server.mjs 的终端查看
```

### 前端连接问题
检查：
```bash
# 1. 确认前端在运行
curl http://localhost:5173

# 2. 检查 API 配置
# WebUI/src/services/api.ts 中的 API_BASE_URL
```

## 当前状态 ✅

- ✅ Miner 运行在端口 8001
- ✅ API Server 运行在端口 3000  
- ✅ Frontend 运行在端口 5173
- ✅ Client 可以正常连接 Miner
- ✅ 所有编译错误已修复
