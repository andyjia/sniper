package fooserver1

import (
	"context"
	"sniper/dao/foodao1"
	pb "sniper/rpc/foo/v1"
	"sniper/util/log"
	"strconv"
)

// Server 实现 /twirp/foo.v1.Foo 服务
type Server struct{}

// Echo 实现 /twirp/foo.v1.Foo/Echo 接口
func (s *Server) Echo(ctx context.Context, req *pb.EchoReq) (resp *pb.EchoResp, err error) {
	query := req.Query
	dao := &foodao1.Dao{}
	result, err := dao.QueryAll(ctx, query)
	if err != nil {
		log.Get(ctx).Errorf("error....=%s", err.Error())
		return &pb.EchoResp{Msg: "NG", Products: nil}, nil
	}
	// return &pb.EchoResp{Msg: req.Msg}, nil
	return &pb.EchoResp{Msg: "OK", Products: result}, nil
}

//Insert service
func (s *Server) Insert(ctx context.Context, req *pb.InsertReq) (resp *pb.InsertResp, err error) {
	dao := &foodao1.Dao{}
	//p := &pb.Product{ProductCode: "1", ProductName: "2", ProductLine: "Planes", ProductScale: "4", ProductVendor: "5", ProductDescription: "6", QuantityInStock: 7, BuyPrice: "11.32", MSRP: "22.33"}
	p := req.Product
	result, err := dao.Insert(ctx, *p)
	if err != nil {
		return &pb.InsertResp{Msg: "NG", InsertRowID: ""}, nil
	}
	rowID, _ := result.LastInsertId()
	return &pb.InsertResp{Msg: "OK", InsertRowID: strconv.FormatInt(rowID, 10)}, nil
}
