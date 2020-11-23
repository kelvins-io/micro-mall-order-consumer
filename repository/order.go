package repository

import (
	"gitee.com/cristiane/micro-mall-order-consumer/model/mysql"
	"gitee.com/kelvins-io/kelvins"
	"xorm.io/xorm"
)

func GetOrderList(sqlSelect, txCode string) ([]mysql.Order, error) {
	var result = make([]mysql.Order, 0)
	err := kelvins.XORM_DBEngine.Table(mysql.TableOrder).Select(sqlSelect).Where("tx_code = ?", txCode).Find(&result)
	return result, err
}

func UpdateOrderByTx(tx *xorm.Session, query, maps interface{}) (int64, error) {
	return tx.Table(mysql.TableOrder).Where(query).Update(maps)
}

func GetOrderCodeList(txCode string) ([]string, error) {
	var collection = make([]mysql.Order, 0)
	var result = make([]string, 0)
	err := kelvins.XORM_DBEngine.Table(mysql.TableOrder).Select("order_code").Where("tx_code = ?", txCode).Find(&collection)
	if err != nil {
		return result, err
	}
	result = make([]string, len(collection))
	for i := 0; i < len(collection); i++ {
		result[i] = collection[i].OrderCode
	}
	return result, err
}
