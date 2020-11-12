package service

import (
	"context"
	"fmt"
	"gitee.com/cristiane/micro-mall-order-consumer/model/args"
	"gitee.com/cristiane/micro-mall-order-consumer/model/mysql"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/code"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/util"
	"gitee.com/cristiane/micro-mall-order-consumer/proto/micro_mall_logistics_proto/logistics_business"
	"gitee.com/cristiane/micro-mall-order-consumer/proto/micro_mall_users_proto/users"
	"gitee.com/cristiane/micro-mall-order-consumer/repository"
	"gitee.com/kelvins-io/common/errcode"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins"
	"time"
)

func TradePayCallbackConsume(ctx context.Context, body string) error {
	fmt.Println("订单支付通知")
	// 通知消息解码
	var businessMsg args.CommonBusinessMsg
	var err error
	err = json.Unmarshal(body, &businessMsg)
	if err != nil {
		kelvins.ErrLogger.Info(ctx, "body:%v Unmarshal err: %v", body, err)
		return err
	}
	if businessMsg.Type != args.TradeOrderEventTypePayCallback {
		return fmt.Errorf(errcode.GetErrMsg(code.NoticeTypeNotEqual))
	}
	var notice args.TradePayCallback
	err = json.Unmarshal(businessMsg.Msg, &notice)
	if err != nil {
		kelvins.ErrLogger.Info(ctx, "businessMsg.Msg: %v Unmarshal err: %v", businessMsg.Msg, err)
		return err
	}
	// 根据订单交易号获取订单
	orderList, err := getOrderListByTxCode(ctx, notice.TxCode)
	if err != nil {
		return err
	}
	// 更新订单状态
	err = updateOrderState(ctx, orderList)
	if err != nil {
		return err
	}
	// 根据交易号获取订单详情
	orderDetailList, err := getOrderDetailListByTxCode(ctx, notice.Uid, notice.TxCode)
	if err != nil {
		return err
	}
	// 处理订单物流
	err = handleOrderLogistics(ctx, orderDetailList)
	if err != nil {
		return err
	}
	return nil
}

func getOrderDetailListByTxCode(ctx context.Context, uid int64, txCode string) ([]args.OrderLogisticsDetail, error) {
	result := make([]args.OrderLogisticsDetail, 0)
	serverName := args.RpcServiceMicroMallUsers
	conn, err := util.GetGrpcClient(serverName)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
		return result, err
	}
	defer conn.Close()
	usersClient := users.NewUsersServiceClient(conn)
	userDeliveryReq := &users.GetUserDeliveryInfoRequest{
		Uid:            uid,
		UserDeliveryId: 0,
	}
	userDeliveryRsp, err := usersClient.GetUserDeliveryInfo(ctx, userDeliveryReq)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetUserDeliveryInfo %v, err: %v, req: %v", serverName, err, userDeliveryReq)
		return result, err
	}
	if userDeliveryRsp.Common.Code != users.RetCode_SUCCESS {
		kelvins.ErrLogger.Errorf(ctx, "GetUserDeliveryInfo %v, err: %v, rsp: %v", serverName, err, userDeliveryRsp)
		return result, err
	}
	deliveryInfo := args.DeliveryLogistics{
		Courier:      "微商城快递",
		CourierType:  1,
		ReceiveType:  1,
		SendTime:     util.ParseTimeOfStr(time.Now().Unix()),
		ReceiveUser:  userDeliveryRsp.Info[0].DeliveryUser,
		ReceiveAddr:  userDeliveryRsp.Info[0].Area + "|" + userDeliveryRsp.Info[0].DetailedArea,
		ReceivePhone: userDeliveryRsp.Info[0].MobilePhone,
	}
	orderToDeliveryInfo := map[string]args.DeliveryLogistics{}
	orderList, err := repository.GetOrderList(txCode)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderCodeList err: %v, TxCode: %v", err, txCode)
		return result, fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}
	orderCodeList := make([]string, len(orderList))
	for i := 0; i < len(orderList); i++ {
		orderCodeList[i] = orderList[i].OrderCode
		deliveryInfo.SendUser = orderList[i].ShopName
		deliveryInfo.SendAddr = orderList[i].ShopAddress
		deliveryInfo.SendTime = util.ParseTimeOfStr(orderList[i].PayExpire.Unix())
		orderToDeliveryInfo[orderList[i].OrderCode] = deliveryInfo
	}
	orderSkuList, err := repository.GetOrderSkuListByOrderCode(orderCodeList)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderSkuListByOrderCode err: %v, orderCodeList: %v", err, orderCodeList)
		return result, fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}
	orderCodeToOrderSku := map[string][]mysql.OrderSku{}
	for i := 0; i < len(orderSkuList); i++ {
		orderCodeToOrderSku[orderSkuList[i].OrderCode] = append(orderCodeToOrderSku[orderSkuList[i].OrderCode], orderSkuList[i])
	}
	for k, _ := range orderCodeToOrderSku {
		goods := make([]args.GoodsLogistics, len(orderCodeToOrderSku[k]))
		goodsList := orderCodeToOrderSku[k]
		for i := 0; i < len(goodsList); i++ {
			goodsLogistics := args.GoodsLogistics{
				SkuCode: goodsList[i].SkuCode,
				Name:    goodsList[i].Name,
				Kind:    goodsList[i].Name,
				Amount:  int64(goodsList[i].Amount),
			}
			goods[i] = goodsLogistics
		}
		detail := args.OrderLogisticsDetail{
			OrderCode: k,
			Goods:     goods,
			Delivery:  orderToDeliveryInfo[k],
		}
		result = append(result, detail)
	}
	return result, nil
}

func getOrderListByTxCode(ctx context.Context, txCode string) ([]mysql.Order, error) {
	// 根据订单交易号获取支付订单
	orderList, err := repository.GetOrderList(txCode)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderList err: %v, TxCode: %v", err, txCode)
		return orderList, fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}
	return orderList, nil
}

func updateOrderState(ctx context.Context, orderList []mysql.Order) error {
	tx := kelvins.XORM_DBEngine.NewSession()
	err := tx.Begin()
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "updateOrderState Begin err: %v", err)
		return fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}
	// 更新订单状态
	for i := 0; i < len(orderList); i++ {
		row := orderList[i]
		where := map[string]interface{}{
			"order_code": row.OrderCode,
		}
		maps := map[string]interface{}{
			"update_time": time.Now(),
			"state":       0,
			"pay_state":   3,
		}
		rowsAffected, err := repository.UpdateOrderByTx(tx, where, maps)
		if err != nil {
			errRollback := tx.Rollback()
			if errRollback != nil {
				kelvins.ErrLogger.Errorf(ctx, "UpdateOrderByTx Rollback err: %v, where: %+v, maps: %+v", errRollback, where, maps)
			}
			kelvins.ErrLogger.Errorf(ctx, "UpdateOrderByTx err: %v, where: %+v, maps: %+v", err, where, maps)
			return fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
		}
		if rowsAffected <= 0 {
			errRollback := tx.Rollback()
			if errRollback != nil {
				kelvins.ErrLogger.Errorf(ctx, "UpdateOrderByTx Rollback err: %v, where: %+v, maps: %+v", errRollback, where, maps)
			}
			return fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
		}
	}
	err = tx.Commit()
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "UpdateOrderByTx Commit err: %v", err)
		return fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}

	return nil
}

func handleOrderLogistics(ctx context.Context, orderList []args.OrderLogisticsDetail) error {
	for i := 0; i < len(orderList); i++ {
		row := orderList[i]
		goods := make([]*logistics_business.GoodsInfo, len(row.Goods))
		for k := 0; k < len(row.Goods); k++ {
			good := &logistics_business.GoodsInfo{
				SkuCode: row.Goods[k].SkuCode,
				Name:    row.Goods[k].Name,
				Kind:    row.Goods[k].Kind,
				Count:   row.Goods[k].Amount,
			}
			goods[k] = good
		}
		reqLogistics := logistics_business.ApplyLogisticsRequest{
			OutTradeNo:  row.OrderCode,
			Courier:     row.Delivery.Courier,
			CourierType: row.Delivery.CourierType,
			ReceiveType: row.Delivery.ReceiveType,
			SendTime:    util.ParseTimeOfStr(time.Now().Unix()),
			Customer: &logistics_business.CustomerInfo{
				SendUser:     row.Delivery.SendUser,
				SendAddr:     row.Delivery.SendAddr,
				SendPhone:    row.Delivery.SendPhone,
				SendTime:     row.Delivery.SendTime,
				ReceiveUser:  row.Delivery.ReceiveUser,
				ReceiveAddr:  row.Delivery.ReceiveAddr,
				ReceivePhone: row.Delivery.ReceivePhone,
			},
			Goods: goods,
		}
		// 关联物流消息
		serverName := args.RpcServiceMicroMallLogistics
		conn, err := util.GetGrpcClient(serverName)
		if err != nil {
			kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
			return err
		}
		clientLogistics := logistics_business.NewLogisticsBusinessServiceClient(conn)
		applyRsp, err := clientLogistics.ApplyLogistics(ctx, &reqLogistics)
		conn.Close()
		if err != nil {
			kelvins.ErrLogger.Errorf(ctx, "ApplyLogistics %v,err: %v, r: %+v", applyRsp, err, reqLogistics)
			return err
		}
		if applyRsp == nil || applyRsp.Common == nil || applyRsp.Common.Code == logistics_business.RetCode_ERROR {
			kelvins.ErrLogger.Errorf(ctx, "ApplyLogistics %v,err: %v, r: %+v", applyRsp, err, reqLogistics)
			return err
		}
		kelvins.BusinessLogger.Infof(ctx, "ApplyLogistics code: %v", applyRsp.LogisticsCode)
	}
	return nil
}

func TradePayCallbackConsumeErr(ctx context.Context, errMsg, body string) error {
	return nil
}
