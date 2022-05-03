package domain

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"regexp"
)

var (
	marketRegex = regexp.MustCompile("[A-Z]{3,4}_[A-Z]{3,4}")
	//	companyCreation, _ = time.Parse(time.RFC3339, "2018-01-01T00:00:00Z00:00")
)

type DealData struct {
	Price        primitive.Decimal128 `json:"price"`
	Volume       primitive.Decimal128 `json:"volume"`
	Market       string               `json:"market"`
	DealId       string               `json:"dealid"`
	IsBuyerMaker bool                 `json:"isbuyermaker"`
}

type Deal struct {
	T    primitive.DateTime `json:"t" bson:"t"`
	Data DealData           `json:"data"`
}

func (d *Deal) Validate() error {
	if d.Data.DealId == "" {
		return errors.New("deal ID is empty")
	}

	if d.Data.Price.IsZero() || d.Data.Price.IsNaN() || d.Data.Price.IsInf() == 1 {
		return errors.Errorf("price value is illegal: %v", d.Data.Price)
	}

	if d.Data.Volume.IsZero() || d.Data.Volume.IsNaN() || d.Data.Volume.IsInf() == 1 {
		return errors.Errorf("volume value is illegal: %v", d.Data.Volume)
	}

	if !marketRegex.MatchString(d.Data.Market) {
		return errors.Errorf(
			"invalid market %s, must be matches by: %s",
			d.Data.Market,
			marketRegex.String(),
		)
	}
	/*
		if d.T.Before(companyCreation) {
			return errors.Errorf("deal created time is to old: %v", d.T)
		}*/

	return nil
}

func MustParseDecimal(num string) primitive.Decimal128 {
	if d, err := primitive.ParseDecimal128(num); err != nil {
		panic(err)
	} else {
		return d
	}
}
