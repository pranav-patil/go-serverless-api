package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"github.com/pranav-patil/go-serverless-api/func/api/middleware"
	"github.com/pranav-patil/go-serverless-api/func/api/routes"
	"github.com/pranav-patil/go-serverless-api/pkg/env"
	"github.com/pranav-patil/go-serverless-api/pkg/logger"
)

var ginLambda *ginadapter.GinLambda

func main() {
	logger.SetGlobalLevel(os.Getenv("LOG_LEVEL"))

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	middleware.Attach(router)
	routes.APIRouter(router)

	if env.IsLocalOrTestEnv() {
		err := router.Run(":8080")

		if err != nil {
			panic(err)
		}
	} else {
		ginLambda = ginadapter.New(router)
		lambda.Start(Handler)
	}
}

func Handler(ctx context.Context, request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return ginLambda.ProxyWithContext(ctx, *request)
}
