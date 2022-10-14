package router

import (
	"account-management/controller"
	"account-management/db"
	"account-management/middlewares"
	"account-management/model"
	"log"

	re "account-management/redis"

	"github.com/gin-gonic/gin"
)

type ApiServer struct {
	*controller.AccountService
}

func InitAPIServer(messageChannels []string) *ApiServer {
	db := db.InitDB()
	accountModel := model.NewAccountModel(db)
	transactionModel := model.NewTransactionModel(db)
	redisClient := re.InitRedisClient()

	accountService := controller.NewAccountService(accountModel, transactionModel, redisClient, messageChannels)

	return &ApiServer{
		AccountService: accountService,
	}
}

func (a *ApiServer) Start() {
	r := gin.Default()
	r.Static("/public", "./public")

	public := r.Group("/api/admin")

	protected := r.Group("/api")

	public.POST("/register", a.Register)
	public.POST("/login", a.Login)

	protected.Use(middlewares.JwtAuthMiddleware())
	protected.GET("/accounts", a.GetAllAccounts)
	protected.POST("/deposit", a.Deposit)
	protected.POST("/withdraw", a.Withdraw)
	protected.POST("/transfer", a.Transfer)
	protected.GET("/transaction/status", a.CheckTransactionStatus)
	protected.GET("/account/balance", a.CheckAccountBalance)

	err := r.Run(":8080") // Ứng dụng chạy tại cổng 8080
	if err != nil {
		log.Fatal(err)
	}
}
