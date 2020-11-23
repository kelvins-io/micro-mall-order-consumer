package repository

import (
	"gitee.com/cristiane/micro-mall-order-consumer/model/mysql"
	"gitee.com/kelvins-io/kelvins"
)

func GetOrderSkuList(sqlSelect string, shopIdList []int64, orderCodeList []string) ([]mysql.OrderSku, error) {
	var result = make([]mysql.OrderSku, 0)
	err := kelvins.XORM_DBEngine.Table(mysql.TableOrderSku).
		Select(sqlSelect).
		In("shop_id", shopIdList).
		In("order_code", orderCodeList).
		Find(&result)
	return result, err
}

func GetOrderSkuListByOrderCode(orderCodeList []string) ([]mysql.OrderSku, error) {
	var result = make([]mysql.OrderSku, 0)
	err := kelvins.XORM_DBEngine.Table(mysql.TableOrderSku).
		In("order_code", orderCodeList).
		Find(&result)
	return result, err
}
