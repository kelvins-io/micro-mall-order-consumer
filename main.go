package main

import (
	"gitee.com/cristiane/micro-mall-order-consumer/startup"
	"gitee.com/cristiane/micro-mall-order-consumer/vars"
	"gitee.com/kelvins-io/kelvins"
	"gitee.com/kelvins-io/kelvins/app"
)

const APP_NAME = "micro-mall-order-consumer"

func main() {
	vars.AppName = APP_NAME
	application := &kelvins.QueueApplication{
		Application: &kelvins.Application{
			LoadConfig: startup.LoadConfig,
			SetupVars:  startup.SetupVars,
			Name:       APP_NAME,
		},
		GetNamedTaskFuncs: startup.GetNamedTaskFuncs,
	}
	app.RunQueueApplication(application)
}
