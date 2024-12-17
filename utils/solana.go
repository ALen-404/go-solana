package main

import (
	"context"
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/blocto/solana-go-sdk/rpc"
	"log"
)

// 创建 Solana 钱包
func CreateWallet() (types.Account, string) {
	// 使用 solana-go-sdk 创建钱包
	account := types.NewAccount()
	// 将钱包的公钥和私钥进行 Base58 编码
	publicKeyBase58 := account.PublicKey.ToBase58()
	privateKeyBase58 := base58.Encode(account.PrivateKey)

	// 打印并返回公钥和私钥
	fmt.Printf("钱包地址: %s\n", publicKeyBase58)
	fmt.Printf("钱包私钥（Base58编码）: %s\n", privateKeyBase58)

	return account, privateKeyBase58
}

// 获取账户余额
func GetBalance(client *client.Client, publicKey string) uint64 {
	// 调用 Solana RPC API 获取账户余额
	balance, err := client.GetBalance(context.Background(), publicKey)
	if err != nil {
		log.Fatalf("获取余额失败: %v", err)
	}
	return balance
}

// main 函数作为程序的入口
func main() {
	// 创建 Solana 钱包
	account, privateKeyBase58 := CreateWallet()

	// 连接到 Solana 测试网（Devnet）
	client := client.NewClient(rpc.DevnetRPCEndpoint) // 使用 Solana Devnet RPC 端点

	// 获取钱包的余额
	balance := GetBalance(client, account.PublicKey.ToBase58()) // 使用 Base58 编码的公钥作为账户地址

	// 打印余额
	fmt.Printf("账户余额：%d lamports\n", balance)
	fmt.Printf("钱包私钥（Base58编码）: %s\n", privateKeyBase58)
}
