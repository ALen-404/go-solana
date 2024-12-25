# Solana交易数据提取工具

该项目用于从Solana区块链提取指定Token（如GOAT和SOL）的交易数据，并将交易记录存储为CSV文件。用户可以设置要提取的最大交易数、RPC地址、Token的Mint地址等。

## 目录结构

```
.
├── cmd
│   └── main.go          // 主程序入口，处理交易提取和CSV写入
├── go.mod               // Go模块管理文件
├── go.sum               // Go模块依赖锁定文件
├── readme.md            // 项目说明文件
└── transactions.csv     // 保存提取到的交易数据
```

## 环境变量配置

程序依赖以下环境变量：

- `RPC_URL`：Solana节点的RPC URL。
- `TOKEN_MINT_ADDRESS`：目标Token的Mint地址。
- `SOL_MINT_ADDRESS`：SOL的Mint地址。
- `GOAT_MINT_ADDRESS`：GOAT的Mint地址。
- `EXCHANGE_ROUTER`：交易所路由地址，用于筛选交易。
- `MAX_TRANSACTIONS`：要提取的最大交易数量。

`.env` 文件示例：

```env
RPC_URL=https://api.mainnet-beta.solana.com
TOKEN_MINT_ADDRESS=YourTokenMintAddress
SOL_MINT_ADDRESS=SolMintAddress
GOAT_MINT_ADDRESS=GoatMintAddress
EXCHANGE_ROUTER=YourExchangeRouter
MAX_TRANSACTIONS=1000
```

## 安装与运行

1. 安装Go环境：请确保您已安装Go 1.18或更高版本。
2. 克隆项目：

   ```bash
   git clone https://github.com/ALen-404/go-solana.git
   cd go-solana
   ```

3. 安装依赖：

   ```bash
   go mod tidy
   ```

4. 配置`.env`文件，确保填入正确的环境变量。
5. 构建并运行程序：

   ```bash
   go run ./cmd/main.go
   ```

运行后，交易数据会以CSV格式保存到`transactions.csv`文件中。

## 功能描述

- **交易数据提取**：从Solana区块链中提取指定Token的交易数据，支持买入（Buy）和卖出（Sell）交易。
- **CSV输出**：提取的数据会保存为CSV文件，包含以下字段：
  - `Date`：交易日期（区块链时间戳）。
  - `Type`：交易类型（Buy 或 Sell）。
  - `GOAT`：GOAT数量变化。
  - `SOL`：SOL数量变化。
  - `Txn`：交易签名。
- **批量处理**：支持分批查询交易签名，并自动处理速率限制。

## 注意事项

- `.env`文件必须正确配置，否则程序无法运行。
- 程序会根据`MAX_TRANSACTIONS`的配置提取最多指定数量的交易数据。
- `transactions.csv`文件会被程序更新，之前的内容会根据模式（追加或覆盖）被处理。

