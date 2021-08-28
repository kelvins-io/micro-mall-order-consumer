package service

import (
	"context"
	"fmt"
	"gitee.com/cristiane/micro-mall-order-consumer/model/args"
	"gitee.com/cristiane/micro-mall-order-consumer/model/mysql"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/code"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/util"
	"gitee.com/cristiane/micro-mall-order-consumer/proto/micro_mall_logistics_proto/logistics_business"
	"gitee.com/cristiane/micro-mall-order-consumer/proto/micro_mall_sku_proto/sku_business"
	"gitee.com/cristiane/micro-mall-order-consumer/proto/micro_mall_users_proto/users"
	"gitee.com/cristiane/micro-mall-order-consumer/repository"
	"gitee.com/kelvins-io/common/errcode"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins"
	"time"
)

func TradePayCallbackConsume(ctx context.Context, body string) error {
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
	time.Sleep(2 * time.Second)
	go func() {
		// 根据订单交易号获取订单
		orderCodeList, err := getOrderListByTxCode(ctx, notice.TxCode)
		if err != nil {
			return
		}
		if len(orderCodeList) == 0 {
			return
		}
		// 更新订单状态
		err = updateOrderState(ctx, orderCodeList)
		if err != nil {
			return
		}
		// 确认订单库存
		err = confirmSkuInventory(ctx, orderCodeList)
		if err != nil {
			return
		}
	}()

	go func() {
		// 根据交易号获取订单详情
		orderDetailList, err := getOrderDetailListByTxCode(ctx, notice.Uid, notice.TxCode)
		if err != nil {
			return
		}
		// 处理订单物流
		err = handleOrderLogistics(ctx, orderDetailList)
		if err != nil {
			return
		}
	}()

	return nil
}

func getOrderDetailListByTxCode(ctx context.Context, uid int64, txCode string) ([]args.OrderLogisticsDetail, error) {
	result := make([]args.OrderLogisticsDetail, 0)
	serverName := args.RpcServiceMicroMallUsers
	conn, err := util.GetGrpcClient(ctx, serverName)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
		return result, err
	}
	//defer conn.Close()
	usersClient := users.NewUsersServiceClient(conn)
	userDeliveryReq := &users.GetUserDeliveryInfoRequest{
		Uid: uid,
	}
	userDeliveryRsp, err := usersClient.GetUserDeliveryInfo(ctx, userDeliveryReq)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetUserDeliveryInfo err: %v, req: %v", err, json.MarshalToStringNoError(userDeliveryReq))
		return result, err
	}
	if userDeliveryRsp.Common.Code != users.RetCode_SUCCESS {
		kelvins.ErrLogger.Errorf(ctx, "GetUserDeliveryInfo err: %v,req: %v rsp: %v", err, json.MarshalToStringNoError(userDeliveryReq), json.MarshalToStringNoError(userDeliveryRsp))
		return result, err
	}
	deliveryInfo := args.DeliveryLogistics{
		Courier:      "微商城快递",
		CourierType:  1,
		ReceiveType:  1,
		SendTime:     util.ParseTimeOfStr(time.Now().Unix()),
		ReceiveUser:  userDeliveryRsp.InfoList[0].DeliveryUser,
		ReceiveAddr:  userDeliveryRsp.InfoList[0].Area + "|" + userDeliveryRsp.InfoList[0].DetailedArea,
		ReceivePhone: userDeliveryRsp.InfoList[0].MobilePhone,
	}
	orderToDeliveryInfo := map[string]args.DeliveryLogistics{}
	orderList, err := repository.GetOrderList("order_code,pay_expire", txCode)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderCodeList err: %v, TxCode: %v", err, txCode)
		return result, fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}
	orderCodeList := make([]string, len(orderList))
	for i := 0; i < len(orderList); i++ {
		orderCodeList[i] = orderList[i].OrderCode
	}
	orderSceneShopList, err := repository.FindOrderSceneShopList(orderCodeList)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "FindOrderSceneShopList err: %v, orderCodeList: %v", err, json.MarshalToStringNoError(orderCodeList))
		return result, fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}
	orderCodeToSceneShop := map[string]mysql.OrderSceneShop{}
	for i := 0; i < len(orderSceneShopList); i++ {
		orderCodeToSceneShop[orderSceneShopList[i].OrderCode] = orderSceneShopList[i]
	}
	for i := 0; i < len(orderList); i++ {
		deliveryInfo.SendUser = orderCodeToSceneShop[orderList[i].OrderCode].ShopName
		deliveryInfo.SendAddr = orderCodeToSceneShop[orderList[i].OrderCode].ShopAddress
		deliveryInfo.SendTime = util.ParseTimeOfStr(orderList[i].PayExpire.Unix())
		orderToDeliveryInfo[orderList[i].OrderCode] = deliveryInfo
	}
	orderSkuList, err := repository.GetOrderSkuListByOrderCode(orderCodeList)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderSkuListByOrderCode err: %v, orderCodeList: %v", err, json.MarshalToStringNoError(orderCodeList))
		return result, fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}
	orderCodeToOrderSku := map[string][]mysql.OrderSku{}
	for i := 0; i < len(orderSkuList); i++ {
		orderCodeToOrderSku[orderSkuList[i].OrderCode] = append(orderCodeToOrderSku[orderSkuList[i].OrderCode], orderSkuList[i])
	}
	for k, _ := range orderCodeToOrderSku {
		goodsList := orderCodeToOrderSku[k]
		goods := make([]args.GoodsLogistics, len(goodsList))
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

func getOrderListByTxCode(ctx context.Context, txCode string) ([]string, error) {
	// 根据订单交易号获取支付订单
	orderList, err := repository.GetOrderList("order_code", txCode)
	orderCodeList := make([]string, len(orderList))
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderList err: %v, TxCode: %v", err, txCode)
		return orderCodeList, fmt.Errorf(errcode.GetErrMsg(code.ErrorServer))
	}
	for i := 0; i < len(orderList); i++ {
		orderCodeList[i] = orderList[i].OrderCode
	}
	return orderCodeList, nil
}

func updateOrderState(ctx context.Context, orderList []string) error {
	// 更新订单状态
	where := map[string]interface{}{
		"order_code": orderList,
	}
	maps := map[string]interface{}{
		"state":            0,
		"pay_state":        3,
		"inventory_verify": 1, // 库存核实
		"update_time":      time.Now(),
	}
	rowsAffected, err := repository.UpdateOrder(where, maps)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "UpdateOrderByTx err: %v, where: %v, maps: %v", err, json.MarshalToStringNoError(where), json.MarshalToStringNoError(maps))
		return err
	}
	_ = rowsAffected

	return nil
}

func confirmSkuInventory(ctx context.Context, orderCodeList []string) error {
	serverName := args.RpcServiceMicroMallSku
	conn, err := util.GetGrpcClient(ctx, serverName)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
		return err
	}
	//defer conn.Close()
	skuClient := sku_business.NewSkuBusinessServiceClient(conn)
	skuReq := &sku_business.ConfirmSkuInventoryRequest{
		OutTradeNo: orderCodeList,
		OpMeta: &sku_business.OperationMeta{
			OpUid: 0,
			OpIp:  "order-consumer-confirm",
		},
	}
	skuRsp, err := skuClient.ConfirmSkuInventory(ctx, skuReq)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "ConfirmSkuInventory err: %v, skuReq: %v", err, json.MarshalToStringNoError(skuReq))
		return err
	}
	if skuRsp.Common.Code != sku_business.RetCode_SUCCESS {
		kelvins.ErrLogger.Errorf(ctx, "ConfirmSkuInventory err: %v, skuReq: %v,skuRsp: %v ", err, json.MarshalToStringNoError(skuReq), json.MarshalToStringNoError(skuRsp))
		return fmt.Errorf("ConfirmSkuInventory err")
	}
	return nil
}

func handleOrderLogistics(ctx context.Context, orderList []args.OrderLogisticsDetail) error {
	// 关联物流消息
	serverName := args.RpcServiceMicroMallLogistics
	conn, err := util.GetGrpcClient(ctx, serverName)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
		return err
	}
	//defer conn.Close()
	clientLogistics := logistics_business.NewLogisticsBusinessServiceClient(conn)
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
		applyRsp, err := clientLogistics.ApplyLogistics(ctx, &reqLogistics)
		if err != nil {
			kelvins.ErrLogger.Errorf(ctx, "ApplyLogistics err: %v, r: %v", err, json.MarshalToStringNoError(reqLogistics))
			return err
		}
		if applyRsp.Common.Code != logistics_business.RetCode_SUCCESS {
			kelvins.ErrLogger.Errorf(ctx, "ApplyLogistics req: %v, resp: %v", json.MarshalToStringNoError(reqLogistics), json.MarshalToStringNoError(applyRsp))
			return err
		}
	}
	return nil
}

func TradePayCallbackConsumeErr(ctx context.Context, errMsg, body string) error {
	return nil
}
