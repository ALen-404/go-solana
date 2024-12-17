package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/mr-tron/base58"
)

// TransactionState 用于管理交易状态
type TransactionState struct {
	Status  string
	Message string
}

// 更新状态
func updateState(state *TransactionState, status, message string) {
	state.Status = status
	state.Message = message
	fmt.Printf("状态: %s, 信息: %s\n", state.Status, state.Message)
}

func main() {
	// 初始化状态管理
	state := &TransactionState{}
	updateState(state, "初始化", "开始执行 Solana 交易程序")

	// 使用给定的私钥（Base58编码）恢复账户
	privateKeyBase58 := "5onJgBRWhMaAqh1bfaY3nuQ3HXaCpkR9aSHZNGLoHX84Egej5jmV9T9JpCQeri6TVUBz5PSftSDMWMbBQCJH3rZ8"
	privateKey, err := base58.Decode(privateKeyBase58)
	if err != nil {
		updateState(state, "错误", fmt.Sprintf("私钥解码失败: %v", err))
		log.Fatalf("私钥解码失败: %v", err)
	}

	// 从私钥字节数组恢复账户
	account, err := types.AccountFromBytes(privateKey)
	if err != nil {
		updateState(state, "错误", fmt.Sprintf("从私钥字节数组恢复账户失败: %v", err))
		log.Fatalf("从私钥字节数组恢复账户失败: %v", err)
	}

	// 输出钱包地址和私钥（Base58编码）
	fmt.Printf("钱包地址: %s\n", account.PublicKey.ToBase58())
	fmt.Printf("钱包私钥（Base58编码）: %s\n", privateKeyBase58)

	// 连接到 Solana 本地测试
	client := client.NewClient("http://127.0.0.1:8899") 
	accountAddress := account.PublicKey.ToBase58()     // 使用给定的公钥作为账户地址

	// 获取账户余额
	balance, err := client.GetBalance(context.Background(), accountAddress)
	if err != nil {
		updateState(state, "错误", fmt.Sprintf("获取余额失败: %v", err))
		log.Fatalf("获取余额失败: %v", err)
	}
	fmt.Printf("账户余额：%d lamports\n", balance)

	// 获取最近的区块哈希（用于构建交易）
	recentBlockhashResponse, err := client.GetLatestBlockhash(context.Background())
	if err != nil {
		updateState(state, "错误", fmt.Sprintf("获取最近区块哈希失败: %v", err))
		log.Fatalf("获取最近区块哈希失败: %v", err)
	}

	// 合约的 ProgramID
	programID := common.PublicKeyFromString("6ZSAnGBubdn1DgHfZ1q3Rigc7gmaN9kX69fLwzTvxH2f") // 替换为你的合约 Program ID

	// 存款金额
	amount := uint64(1000)
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, amount)

	// 创建合约指令：调用合约进行存款
	txInstruction := types.Instruction{
		ProgramID: programID, // 合约地址
		Accounts: []types.AccountMeta{
			{
				PubKey:     account.PublicKey,  // 发送账户的公钥
				IsSigner:   true,               // 需要签名
				IsWritable: true,               // 可写
			},
		},
		Data: data, // 合约的输入数据（存款金额）
	}

	// 创建交易消息
	message := types.NewMessage(types.NewMessageParam{
		FeePayer:        account.PublicKey,             // 费用支付者
		RecentBlockhash: recentBlockhashResponse.Blockhash, // 最新的区块哈希
		Instructions: []types.Instruction{
			txInstruction, // 使用与合约交互的指令
		},
	})

	// 创建交易对象
	tx, err := types.NewTransaction(types.NewTransactionParam{
		Signers: []types.Account{account}, // 只需要一个签名者
		Message: message,
	})
	if err != nil {
		updateState(state, "错误", fmt.Sprintf("创建交易失败: %v", err))
		log.Fatalf("创建交易失败: %v", err)
	}

	// 发送交易
	txhash, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		updateState(state, "错误", fmt.Sprintf("发送交易失败: %v", err))
		log.Fatalf("发送交易失败: %v", err)
	}
	updateState(state, "成功", fmt.Sprintf("交易成功，交易哈希: %s", txhash))
}
