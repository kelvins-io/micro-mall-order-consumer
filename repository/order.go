package repository

import (
	"gitee.com/cristiane/micro-mall-order-consumer/model/mysql"
	"gitee.com/kelvins-io/kelvins"
)

func GetOrderList(txCode string) ([]mysql.Order, error) {
	var result = make([]mysql.Order, 0)
	err := kelvins.XORM_DBEngine.Table(mysql.TableOrder).Where("tx_code = ?", txCode).Find(&result)
	return result, err
}
