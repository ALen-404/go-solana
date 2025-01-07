package main

import (
	"context"
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/ALen-404/go-solana/raydium"
	"github.com/ALen-404/go-solana/raydium/trade"
	"github.com/ALen-404/go-solana/raydium/utils"
)

// swapTokens 执行代币交换操作
func swapTokens(rpcURL, privateKeyBase58, inputTokenSymbol, inputMint, outputTokenSymbol, outputMint string, amountIn float64, slippagePercent uint64) (string, error) {
	// 创建与Solana网络的连接
	connection := rpc.New(rpcURL)

	// 创建Raydium客户端
	raydium := raydium.New(connection, privateKeyBase58)

	// 设置输入和输出代币
	fmt.Println("Input Token Symbol:", inputTokenSymbol)
	fmt.Println("Input Token Mint:", inputMint)
	fmt.Println("Output Token Symbol:", outputTokenSymbol)
	fmt.Println("Output Token Mint:", outputMint)

	inputToken := utils.NewToken(inputTokenSymbol, inputMint, 9)
	outputToken := utils.NewToken(outputTokenSymbol, outputMint, 6)

	// 设置滑点
	slippage := utils.NewPercent(slippagePercent, 100)

	// 设置交易金额
	amount := utils.NewTokenAmount(inputToken, amountIn)

	// 获取池的键值
	poolKeys, err := raydium.Pool.GetPoolKeys(inputToken.Mint, outputToken.Mint)
	if err != nil {
		return "", fmt.Errorf("failed to get pool keys: %w", err)
	}

	// 输出池键值信息，检查是否成功获取池信息
	fmt.Println("Pool Keys:", poolKeys)

	// 计算交换后的输出金额
	amountsOut, err := raydium.Liquidity.GetAmountsOut(poolKeys, amount, slippage)
	if err != nil {
		return "", fmt.Errorf("failed to get amounts out: %w", err)
	}

	// 构建交换交易
	tx, err := raydium.Trade.MakeSwapTransaction(
		poolKeys,
		amountsOut.AmountIn,
		amountsOut.MinAmountOut,
		trade.FeeConfig{
			MicroLamports: 25000, // 0.000025 SOL手续费
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create swap transaction: %w", err)
	}

	// 模拟交易
	simRes, err := connection.SimulateTransaction(context.Background(), tx)
	if err != nil {
		return "", fmt.Errorf("transaction simulation failed: %w", err)
	}

	// 输出模拟结果
	spew.Dump(simRes)

	// 如果一切正常，发送交易
	signature, err := connection.SendTransactionWithOpts(context.Background(), tx, rpc.TransactionOpts{SkipPreflight: true})
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	// 返回交易签名，使用 .String() 方法将 signature 转换为字符串
	return signature.String(), nil
}

func main() {
	// 设置RPC URL和私钥（直接写在代码中）
	rpcURL := "https://broken-muddy-butterfly.solana-mainnet.quiknode.pro/270ff8923ae3fcd2e905cf2dd38c6f379a317cca"
	privateKeyBytes := []byte{
		156, 186, 118, 227, 248, 14, 43, 163, 24, 250, 116, 42, 5, 35, 177, 124, 103, 158, 139, 94, 124, 14, 172, 12,
		220, 230, 105, 121, 95, 75, 82, 89, 108, 102, 152, 114, 216, 222, 129, 39, 142, 152, 62, 42, 195, 107, 195,
		111, 191, 189, 216, 159, 183, 136, 57, 35, 20, 170, 171, 227, 19, 211, 235, 215,
	}
	// 将字节数组转换为 Base58 编码的字符串
	privateKeyBase58 := base58.Encode(privateKeyBytes)

	// 设置交换的代币参数
	inputTokenSymbol := "SOL"
	inputMint := "So11111111111111111111111111111111111111112"
	outputTokenSymbol := "RAY"
	outputMint := "4k3Dyjzvzp8eMZWUXbBCjEvwSkkk59S5iCNLY3QrkX6R"
	amountIn := 0.01  // 输入的SOL数量
	slippagePercent := uint64(1) // 设置滑点为1%

	// 执行交换操作
	signature, err := swapTokens(rpcURL, privateKeyBase58, inputTokenSymbol, inputMint, outputTokenSymbol, outputMint, amountIn, slippagePercent)
	if err != nil {
		fmt.Println("Error executing swap:", err)
		return
	}

	// 输出交易签名
	fmt.Println("Transaction successfully sent with signature:", signature)
}
