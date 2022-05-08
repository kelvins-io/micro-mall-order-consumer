package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitee.com/cristiane/micro-mall-order-consumer/model/args"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/code"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/util"
	"gitee.com/cristiane/micro-mall-order-consumer/pkg/util/email"
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
	err = json.Unmarshal(businessMsg.Content, &notice)
	if err != nil {
		kelvins.ErrLogger.Info(ctx, "businessMsg.Msg: %v Unmarshal err: %v", businessMsg.Content, err)
		return err
	}
	time.Sleep(3 * time.Second) // 订单创建事务先执行
	// 从数据库查询订单涉及的商品
	orderList, err := repository.GetOrderList("shop_id,order_code,amount", notice.TxCode)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderList err: %v, tx_code: %v", err, notice.TxCode)
		return err
	}
	if len(orderList) == 0 {
		return nil
	}
	shopIdList := make([]int64, len(orderList))
	orderCodeList := make([]string, len(orderList))
	skuNotice := strings.Builder{}
	for i := 0; i < len(orderList); i++ {
		skuNotice.WriteString(fmt.Sprintf("【店铺ID：%v，订单号：%v，关联%v份商品】", orderList[i].ShopId, orderList[i].OrderCode, orderList[i].Amount))
		shopIdList[i] = orderList[i].ShopId
		orderCodeList[i] = orderList[i].OrderCode
	}
	orderSkuList, err := repository.GetOrderSkuList("shop_id,sku_code", shopIdList, orderCodeList)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetOrderSkuList ,err: %v, shopIdList: %v,orderCodeList: %v", err, json.MarshalToStringNoError(shopIdList), json.MarshalToStringNoError(orderCodeList))
		return err
	}
	if len(orderSkuList) == 0 {
		return nil
	}

	// 邮件通知
	go func() {
		userInfo, err := getUserInfo(ctx, notice.Uid)
		if err != nil {
			kelvins.ErrLogger.Errorf(ctx, "getUserInfo ,err: %v, uid: %v", err, notice.Uid)
			return
		}
		if userInfo != nil && userInfo.Email != "" {
			emailNotice := fmt.Sprintf(args.UserCreateOrderTemplate, userInfo.UserName, util.ParseTimeOfStr(time.Now().Unix()), skuNotice.String())
			err := email.SendEmailNotice(ctx, userInfo.Email, kelvins.AppName, emailNotice)
			if err != nil {
				kelvins.ErrLogger.Info(ctx, "noticeUserPayResult SendEmailNotice err, emailNotice: %v", emailNotice)
			}
		}
	}()

	// 从购物车中删除商品
	go func() {
		for i := 0; i < len(orderSkuList); i++ {
			orderSku := orderSkuList[i]
			serverName := args.RpcServiceMicroMallUserTrolley
			conn, err := util.GetGrpcClient(ctx, serverName)
			if err != nil {
				kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
				return
			}
			//defer conn.Close()
			client := trolley_business.NewTrolleyBusinessServiceClient(conn)
			r := trolley_business.RemoveSkuRequest{
				Uid:     notice.Uid,
				ShopId:  orderSku.ShopId,
				SkuCode: orderSku.SkuCode,
				Count:   -1, // 全部移除
			}
			rsp, err := client.RemoveSku(ctx, &r)
			if err != nil {
				kelvins.ErrLogger.Errorf(ctx, "RemoveSku %v,err: %v, r: %v", serverName, err, json.MarshalToStringNoError(r))
				return
			}
			if rsp.Common.Code != trolley_business.RetCode_SUCCESS {
				kelvins.ErrLogger.Errorf(ctx, "RemoveSku req %v, rsp: %v", json.MarshalToStringNoError(r), json.MarshalToStringNoError(rsp))
				return
			}
		}
	}()

	return nil
}

func getUserInfo(ctx context.Context, uid int64) (userInfo *users.UserInfo, err error) {
	serverName := args.RpcServiceMicroMallUsers
	conn, err := util.GetGrpcClient(ctx, serverName)
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetGrpcClient %v,err: %v", serverName, err)
		return
	}
	client := users.NewUsersServiceClient(conn)
	result, err := client.GetUserInfo(ctx, &users.GetUserInfoRequest{Uid: uid})
	if err != nil {
		kelvins.ErrLogger.Errorf(ctx, "GetUserInfo err: %v, uid: %v", err, uid)
		return
	}
	if result.Common.Code == users.RetCode_SUCCESS {
		userInfo = result.GetInfo()
	}

	return
}

func TradeOrderConsumeErr(ctx context.Context, errMsg, body string) error {
	return nil
}
