package service

import (
	"account-management/db.go"
	"account-management/model"
	re "account-management/redis"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

var (
	// channel     = "mychannel3"
	channelList = []string{"deposit", "withdraw", "transfer"}
	// numOfWorkers           = 1
	minimumBalance float64 = 50000
)

func StartTaskQueue() {
	db := db.InitDB()
	accountModel := model.NewAccountModel(db)
	transactionModel := model.NewTransactionModel(db)
	rdb := re.InitRedisClient()

	for _, channel := range channelList {
		go func(channel string) {
			subscriber := rdb.Subscribe(channel)
			// defer subscriber.Close()

			if err := subscriber.Ping(); err != nil {
				log.Fatalln("Redis server is busy !")
			}

			// messageChannel := subscriber.Channel()

			for {
				message, err := subscriber.ReceiveMessage()
				if err != nil {
					log.Fatal(err)
				}

				err = Worker(rdb, message, accountModel, transactionModel)
				if err != nil {
					fmt.Println(err)
				}
			}
		}(channel)

	}

	time.Sleep(time.Hour * 10000)

}

func Worker(rdb *redis.Client, message *redis.Message, accountModel *model.AccountModel, transactionModel *model.TransactionModel) error {

	payload := message.Payload

	var tx model.Transaction

	err := json.Unmarshal([]byte(payload), &tx)
	if err != nil {
		return err
	}

	fmt.Printf("Worker is processing %s request of User %s \n", tx.Type, tx.Sender)

	err = ProcessTransaction(accountModel, transactionModel, &tx)
	if err != nil {
		return err
	}

	return nil

}

func ProcessTransaction(accountModel *model.AccountModel, transactionModel *model.TransactionModel, tx *model.Transaction) error {

	senderAmount, err := accountModel.GetAccountBalance(tx.Sender)
	if err != nil {
		return err
	}

	var newSenderBalance float64
	var newReceiverBalance float64

	if tx.Type == "Deposit" {
		newSenderBalance = senderAmount + tx.Amount
	} else {
		if senderAmount-tx.Amount < minimumBalance {
			return fmt.Errorf("your balance is not enough to %s", strings.ToLower(tx.Type))
		}

		newSenderBalance = senderAmount - tx.Amount
	}

	if tx.Type == "Transfer" {
		receiverAmount, err := accountModel.GetAccountBalance(tx.Receiver)
		if err != nil {
			return err
		}

		newReceiverBalance = receiverAmount + tx.Amount
	}

	err = accountModel.SaveNewBalance(newSenderBalance, tx.Sender)
	if err != nil {
		return err
	}

	if tx.Type == "Transfer" {
		err = accountModel.SaveNewBalance(newReceiverBalance, tx.Receiver)
		if err != nil {
			return err
		}
	}

	err = transactionModel.Save(tx)
	if err != nil {
		return err
	}

	return nil
}
