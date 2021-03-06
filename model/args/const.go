package args

type MerchantsMaterialInfo struct {
	Uid          int64
	MaterialId   int64
	RegisterAddr string
	HealthCardNo string
	Identity     int32
	State        int32
	TaxCardNo    string
}

type DeliveryLogistics struct {
	Courier       string `json:"courier"`
	CourierType   int32  `json:"courier_type"`
	ReceiveType   int32  `json:"receive_type"`
	SendUser      string `json:"send_user"`
	SendUserId    int64  `json:"send_user_id"`
	SendAddr      string `json:"send_addr"`
	SendPhone     string `json:"send_phone"`
	SendTime      string `json:"send_time"`
	ReceiveUser   string `json:"receive_user"`
	ReceiveAddr   string `json:"receive_addr"`
	ReceivePhone  string `json:"receive_phone"`
	ReceiveUserId int64  `json:"receive_user_id"`
}

type OrderLogisticsDetail struct {
	OrderCode string            `json:"order_code"`
	Goods     []GoodsLogistics  `json:"goods"`
	Delivery  DeliveryLogistics `json:"delivery"`
}

type GoodsLogistics struct {
	SkuCode string `json:"sku_code"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Amount  int64  `json:"amount"`
}

const (
	RpcServiceMicroMallUsers       = "micro-mall-users"
	RpcServiceMicroMallSku         = "micro-mall-sku"
	RpcServiceMicroMallLogistics   = "micro-mall-logistics"
	RpcServiceMicroMallUserTrolley = "micro-mall-trolley"
)

const (
	UserCreateOrderTemplate = "尊敬的用户【%v】你好，你于：%v 购买的 %v 已经成功下单了，请尽快支付哟"
)

type CommonBusinessMsg struct {
	Type    int    `json:"type"`
	Tag     string `json:"tag"`
	UUID    string `json:"uuid"`
	Content string `json:"content"`
}

type TradeOrderDetail struct {
	ShopId    int64  `json:"shop_id"`
	OrderCode string `json:"order_code"`
}

type TradeOrderNotice struct {
	Uid    int64  `json:"uid"`
	Time   string `json:"time"`
	TxCode string `json:"tx_code"`
}

type TradePayCallback struct {
	Uid    int64  `json:"uid"`
	TxCode string `json:"tx_code"`
	PayId  string `json:"pay_id"`
}

const (
	Unknown                        = 0
	TradeOrderEventTypeCreate      = 10014
	TradeOrderEventTypeExpire      = 10015
	TradeOrderEventTypePayCallback = 10018
)

var MsgFlags = map[int]string{
	Unknown:                        "未知",
	TradeOrderEventTypeCreate:      "交易订单创建",
	TradeOrderEventTypeExpire:      "交易订单过期",
	TradeOrderEventTypePayCallback: "交易订单支付回调",
}

func GetMsg(code int) string {
	msg, ok := MsgFlags[code]
	if ok {
		return msg
	}
	return MsgFlags[Unknown]
}
