package repository

import (
	"gitee.com/cristiane/micro-mall-order-consumer/model/mysql"
	"gitee.com/kelvins-io/kelvins"
)

func FindOrderSceneShopList(orderCodes []string) ([]mysql.OrderSceneShop, error) {
	var result = make([]mysql.OrderSceneShop, 0)
	err := kelvins.XORM_DBEngine.Table(mysql.TableOrderSceneShop).
		In("order_code", orderCodes).
		Find(&result)
	return result, err
}
