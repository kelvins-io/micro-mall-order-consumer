package service

import (
	"context"
	"fmt"
	"gitee.com/cristiane/micro-mall-order-consumer/model/args"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/code"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/util"
	"gitee.com/cristiane/micro-mall-order-consumer/proto/micro_mall_trolley_proto/trolley_business"
	"gitee.com/cristiane/micro-mall-order-consumer/proto/micro_mall_users_proto/users"
	"gitee.com/cristiane/micro-mall-order-consumer/repository"
	"gitee.com/kelvins-io/common/errcode"
	"gitee.com/kelvins-io/common/json"
	"gitee.com/kelvins-io/kelvins"
)

func TradeOrderConsume(ctx context.Context, body string) error {
	// 通知消息解码
	var businessMsg args.CommonBusinessMsg
	var err error
	err = json.Unmarshal(body, &businessMsg)
	if err != nil {
		kelvins.ErrLogger.Info(ctx, "body:%v Unmarshal err: %v", body, err)
		return err
	}
	if businessMsg.Type != args.TradeOrderEventTypeCreate {
		return fmt.Errorf(errcode.GetErrMsg(code.NoticeTypeNotEqual))
	}
	var notice args.TradeOrderNotice
	err = json.Unmarshal(businessMsg.Msg, &notice)
	if err != nil {
		kelvins.ErrLogger.Info(ctx, "businessMsg.Msg: %v Unmarshal err: %v", businessMsg.Msg, err)
		return err
	}
	// 获取用户信息
	serverName := args.RpcServiceMicroMallUsers
	conn, err := util.GetGrpcClient(serverName)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
		return err
	}
	defer conn.Close()
	client := users.NewUsersServiceClient(conn)
	r := users.GetUserInfoRequest{
		Uid: notice.Uid,
	}
	rsp, err := client.GetUserInfo(ctx, &r)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetUserInfo %v,err: %v, r: %+v", serverName, err, r)
		return err
	}
	if rsp.Common.Code != users.RetCode_SUCCESS {
		kelvins.ErrLogger.Errorf(ctx, "GetUserInfo %v,not ok : %v, rsp: %+v", serverName, err, rsp)
		return fmt.Errorf(rsp.Common.Msg)
	}
	if rsp.Info == nil || rsp.Info.AccountId == "" {
		kelvins.ErrLogger.Errorf(ctx, "GetUserInfo %v,accountId null : %v, rsp: %+v", serverName, err, rsp)
		return fmt.Errorf(errcode.GetErrMsg(code.UserNotExist))
	}

	// 从数据库查询订单涉及的商品
	orderList, err := repository.GetOrderList("shop_id,order_code", notice.TxCode)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderList err: %v, tx_code: %v", err, notice.TxCode)
		return err
	}
	shopIdList := make([]int64, len(orderList))
	orderCodeList := make([]string, len(orderList))
	for i := 0; i < len(orderList); i++ {
		shopIdList[i] = orderList[i].ShopId
		orderCodeList[i] = orderList[i].OrderCode
	}
	orderSkuList, err := repository.GetOrderSkuList("shop_id,sku_code", shopIdList, orderCodeList)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderSkuList ,err: %v, shopIdList: %+v,orderCodeList: %+v", err, shopIdList, orderCodeList)
		return err
	}
	// 从购物车中删除商品
	for i := 0; i < len(orderSkuList); i++ {
		orderSku := orderSkuList[i]
		serverName := args.RpcServiceMicroMallUserTrolley
		conn, err := util.GetGrpcClient(serverName)
		if err != nil {
			kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
			return err
		}
		client := trolley_business.NewTrolleyBusinessServiceClient(conn)
		r := trolley_business.RemoveSkuRequest{
			Uid:     notice.Uid,
			ShopId:  orderSku.ShopId,
			SkuCode: orderSku.SkuCode,
		}
		rsp, err := client.RemoveSku(ctx, &r)
		if err != nil {
			_ = conn.Close()
			kelvins.ErrLogger.Errorf(ctx, "RemoveSku %v,err: %v, r: %+v", serverName, err, r)
			return err
		}
		if rsp.Common.Code != trolley_business.RetCode_SUCCESS {
			_ = conn.Close()
			kelvins.ErrLogger.Errorf(ctx, "RemoveSku %v,not ok : %v, rsp: %+v", serverName, err, rsp)
			return fmt.Errorf(rsp.Common.Msg)
		}
		_ = conn.Close()
	}

	return nil
}

func TradeOrderConsumeErr(ctx context.Context, errMsg, body string) error {
	return nil
}
