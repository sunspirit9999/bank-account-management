package router

import (
	"account-management/controller"
	"account-management/db"
	"account-management/middlewares"
	"account-management/model"
	re "account-management/redis"
	"log"

	"github.com/gin-gonic/gin"
)

func Start() {
	r := gin.Default()
	r.Static("/public", "./public")

	public := r.Group("/api/admin")

	protected := r.Group("/api")

	db := db.InitDB()
	accountModel := model.NewAccountModel(db)
	transactionModel := model.NewTransactionModel(db)
	redisClient := re.InitRedisClient()
	accountService := controller.NewAccountService(accountModel, transactionModel, redisClient)

	public.POST("/register", accountService.Register)
	public.POST("/login", accountService.Login)

	protected.Use(middlewares.JwtAuthMiddleware())
	protected.GET("/accounts", accountService.GetAllAccounts)
	protected.POST("/deposit", accountService.Deposit)
	protected.POST("/withdraw", accountService.Withdraw)
	protected.POST("/transfer", accountService.Transfer)
	protected.GET("/transaction/status", accountService.CheckTransactionStatus)
	protected.GET("/account/balance", accountService.CheckAccountBalance)

	err := r.Run(":8080") // Ứng dụng chạy tại cổng 8080
	if err != nil {
		log.Fatal(err)
	}
}
