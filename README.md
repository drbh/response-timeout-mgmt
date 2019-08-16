# timouter 


A easy way to set a timeout for an event, made to be able to handle concurrent updates.

```go
package main

import (
	"fmt"
	timouter "github.com/drbh/response-timeout-mgmt"
	"github.com/gin-gonic/gin"
)

func main() {

	u := "USERID-001"

	messageReminder := timouter.GetInstance()

	messageReminder.AddUser(u, 10)
	messageReminder.AddCallback(u, func(usm UserStateMachine) {
		if !usm.WhenRecieved.IsZero() {
			fmt.Println("They answered in time!")
		} else {
			fmt.Println("They didnt answer in time")
		}
	})
	messageReminder.GetUserAndTriggerEvent(u, "init")
	messageReminder.GetUserAndTriggerEvent(u, "send_message")

	r := gin.Default()
	r.GET("/user_response", func(c *gin.Context) {

		messageReminder.GetUserAndTriggerEvent(u, "user_response")
		messageReminder.GetUserAndTriggerEvent(u, "done")

		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.Run(":9000")
}
```