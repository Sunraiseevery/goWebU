# goWebU

[English](README.md) | [中文](README.zh.md)

简单的 SSH 端口转发服务，使用 SQLite 存储主机记录和命令历史。

## 构建

```
go build
```

## 运行

```
./goWebU -db data.db -addr :8080
```

或直接运行而无需构建：

```
go run . -db data.db -addr :8080
```

服务器启动后会尝试打开浏览器访问 `http://localhost:8080/`，如果没有自动打开，可以手动访问该地址。

## Web 界面

服务器提供一个简单的静态界面，用于管理主机并启动隧道。

### API 接口

- `GET /hosts` 列出保存的主机
- `POST /hosts` 添加或更新主机
- `POST /start` 启动新隧道（记录命令历史）
- `POST /stop` 停止运行中的隧道
- `GET /history` 列出最近的命令历史
