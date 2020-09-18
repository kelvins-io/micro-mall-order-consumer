package mysql

import (
	"time"
)

const (
	TableOrderSku = "order_sku"
)

type OrderSku struct {
	Id         int64     `xorm:"pk autoincr comment('自增ID') BIGINT"`
	OrderCode  string    `xorm:"not null comment('对应订单code') unique(order_unique) CHAR(64)"`
	ShopId     int64     `xorm:"not null comment('店铺ID') unique(order_unique) BIGINT"`
	SkuCode    string    `xorm:"not null comment('商品sku') unique(order_unique) CHAR(64)"`
	Price      string    `xorm:"not null default 0.0000000000000000 comment('商品单价') DECIMAL(32,16)"`
	Amount     int       `xorm:"not null comment('商品数量') INT"`
	Name       string    `xorm:"comment('商品名称') index VARCHAR(255)"`
	CoinType   int       `xorm:"not null default 1 comment('币种，1-cny,2-usd') TINYINT"`
	CreateTime time.Time `xorm:"not null default CURRENT_TIMESTAMP comment('创建时间') DATETIME"`
	UpdateTime time.Time `xorm:"not null default CURRENT_TIMESTAMP comment('修改时间') DATETIME"`
}
