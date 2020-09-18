package startup

import "gitee.com/cristiane/micro-mall-order-consumer/service"

const (
	TaskNameTradeOrderNotice    = "task_trade_order_notice"
	TaskNameTradeOrderNoticeErr = "task_trade_order_notice_err"
)

func GetNamedTaskFuncs() map[string]interface{} {

	var taskRegister = map[string]interface{}{
		TaskNameTradeOrderNotice:    service.TradeOrderConsume,
		TaskNameTradeOrderNoticeErr: service.TradeOrderConsumeErr,
	}
	return taskRegister
}
