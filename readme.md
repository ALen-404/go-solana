# Go-Solana

基于 Go 的应用程序，用于从 Solana 区块链获取交易数据、处理数据并将其保存到 CSV 文件中。

## 功能
- 从 Solana 地址获取交易数据。
- 处理交易以计算 SOL 和 GOAT 余额变化。
- 将交易详情保存到 CSV 文件中。
- 支持速率限制以避免过多请求 RPC。
- 可通过环境变量进行配置。

## 文件结构
```
.
├── cmd
│   └── main.go         # 主程序逻辑
├── go.mod              # Go 模块定义
├── go.sum              # Go 依赖文件
├── readme.md           # 项目文档
└── transactions.csv    # 输出的已处理交易数据文件
```

## 安装

1. 克隆代码库：
   ```bash
   git clone https://github.com/ALen-404/go-solana.git
   cd go-solana
   ```

2. 安装依赖：
   ```bash
   go mod tidy
   ```

3. 在项目根目录下创建一个 `.env` 文件，并配置以下环境变量：
   ```env
   RPC_URL=<你的-Solana-RPC-URL>
   TOKEN_MINT_ADDRESS=<你的-Token-Mint-地址>
   SOL_MINT_ADDRESS=<你的-SOL-Mint-地址>
   GOAT_MINT_ADDRESS=<你的-GOAT-Mint-地址>
   EXCHANGE_ROUTER=<你的-Exchange-Router-地址>
   MAX_TRANSACTIONS=<最大交易数量>
   ```

## 使用方法

1. 运行程序：
   ```bash
   go run cmd/main.go
   ```

2. 程序将会：
   - 获取指定 Token 地址的交易签名。
   - 处理每笔交易以计算 SOL 和 GOAT 的余额变化。
   - 将处理后的交易数据保存到 `transactions.csv` 文件中。

## 交易数据格式
处理后的交易数据将保存到 `transactions.csv` 文件中，具有以下列：

| 日期                | 时间戳         | 类型  | GOAT       | SOL         | 交易签名    |
|---------------------|----------------|-------|------------|-------------|------------|
| 交易日期            | Unix 时间戳    | 买/卖 | GOAT 数量  | SOL 数量    | 交易签名    |

## 核心功能
- **`writeTransactionsToCSV`**: 将交易数据写入 CSV 文件。
- **`roundTo6Decimal`**: 将浮点数保留到小数点后六位。
- **`roundTo9Decimal`**: 将浮点数保留到小数点后九位。

## 依赖
- [solana-go-sdk](https://github.com/blocto/solana-go-sdk): 用于 Go 的 Solana SDK。
- [godotenv](https://github.com/joho/godotenv): 从 `.env` 文件加载环境变量。
- [rate](https://pkg.go.dev/golang.org/x/time/rate): 速率限制工具。

## 注意事项
- 确保你的 Solana RPC URL 是活动的，并且有足够的吞吐量来处理请求。
- 程序会跳过带有错误或失败状态的交易。

