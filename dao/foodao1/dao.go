package foodao1

import (
	"context"
	"database/sql"
	pb "sniper/rpc/foo/v1"
	"sniper/util/db"
	"sniper/util/log"
)

// Dao implement DB oper
type Dao struct{}

// QueryAll query all records
func (d *Dao) QueryAll(ctx context.Context, queryString string) (product []*pb.Product, err error) {
	var products []*pb.Product

	c := db.Get(ctx, "default")
	log.Get(ctx).Infoln("queryString", queryString)
	sql := "select productCode, productName from products where " + queryString
	q := db.SQLSelect("products", sql)
	result, err := c.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}

	for result.Next() {
		product := pb.Product{}
		err := result.Scan(&product.ProductCode, &product.ProductName)
		if err != nil {
			return nil, err
		}
		products = append(products, &product)
	}

	return products, nil
}

//Insert a new product
func (d *Dao) Insert(ctx context.Context, p pb.Product) (result sql.Result, err error) {
	c := db.Get(ctx, "default")
	sql := "insert into products values(?,?,?,?,?,?,?,?,?)"
	q := db.SQLInsert("products", sql)
	result, err = c.ExecContext(ctx, q, p.ProductCode, p.ProductName, p.ProductLine, p.ProductScale, p.ProductVendor, p.ProductDescription, p.QuantityInStock, p.BuyPrice, p.MSRP)
	if err != nil {
		log.Get(ctx).Errorf("*****insert error=%s\n", err.Error())
		return nil, err
	}

	return result, nil

}
