# Solana 钱包生成器和连接器

## 项目概述

这个项目是一个用 Go 语言编写的 Solana 钱包工具。它可以批量生成 Solana 钱包，并提供与 Solana 网络连接的功能。主要特性包括：

1. 批量生成 Solana 钱包（包括公钥和私钥）
2. 连接到 Solana 网络（默认使用 Devnet）
3. 查询钱包信息（如余额和账户数据）
4. 尝试为新生成的钱包请求 SOL 空投（在 Devnet 上）

## 安装要求

- Go 1.22 或更高版本
- Solana CLI（用于空投功能）

## 安装步骤

1. 克隆仓库：
``` git clone https://github.com/ALen-404/go-solana.git```
``` cd go-solana```
2. 安装依赖：
``` go mod tidy```
3. 确保 Solana CLI 已正确安装并配置为使用 Devnet：
```solana config set --url https://api.devnet.solana.com```

## 使用方法

运行主程序：
``` 
go run cmd/main/main.go
```

这将执行以下操作：

1. 生成多个 Solana 钱包
2. 打印每个钱包的公钥和私钥
3. 保存第一个钱包的公钥到文件
4. 尝试为第一个钱包请求 SOL 空投
5. 查询并显示第一个钱包的信息

## 项目结构

```
.
├── airdrop.sh
├── cmd
│   └── main
│       └── main.go
├── go.mod
├── go.sum
├── internal
│   └── wallet
│       ├── connect.go
│       └── generate.go
└── README.md    
```

## 注意事项

1. **Devnet 限制**：
    - Devnet 上的空投有速率限制。如果遇到错误，请稍后再试或使用在线水龙头 [Solana Faucet](https://solfaucet.com/) 获取测试用 SOL。
    - Devnet 上的 SOL 没有实际价值，仅用于测试目的。

2. **私钥安全性**：
    - 程序会直接打印私钥，仅用于演示目的。在生产环境中，请妥善保管私钥，避免泄露。

3. **主网配置**：
    - 如果需要连接主网，请修改代码中的 RPC URL 为 Mainnet-beta 的地址，例如：
      ```
      const MainnetRPCEndpoint = "https://api.mainnet-beta.solana.com"
      ```

4. **本地测试账本**：
    - 如果您运行了本地验证器（test-ledger），请确保正确配置 RPC URL 并启动验证器服务。

## 常见问题

### Q1: 为什么空投失败？
A: Devnet 上空投可能因速率限制而失败。请稍等片刻后重试，或者使用在线水龙头获取 SOL。

### Q2: 如何增加生成的钱包数量？
A: 修改 `cmd/main/main.go` 中调用 `GenerateMultipleWallets` 的参数。例如，将 `5` 改为 `10`：
```
wallets, err := wallet.GenerateMultipleWallets(10)
```

### Q3: 如何切换到主网？
A: 修改 `internal/wallet/connect.go` 中的 RPC URL 为主网地址，并确保您的钱包中有真实的 SOL。

