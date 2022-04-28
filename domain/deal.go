package domain

import (
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var marketRegex = regexp.MustCompile("[A-Z]{3,4}_[A-Z]{3,4}")
var companyCreation, _ = time.Parse(time.RFC3339, "2018-01-01T00:00:00Z00:00")

type Deal struct {
	ID           primitive.ObjectID   `json:"_id"`
	Price        primitive.Decimal128 `json:"price"`
	Volume       primitive.Decimal128 `json:"volume"`
	Time         time.Time            `json:"time"`
	Market       string               `json:"market"`
	DealId       string               `json:"dealid"`
	IsBuyerMaker bool                 `json:"isbuyermaker"`
}

func (d *Deal) Validate() error {
	if d.DealId == "" {
		return errors.New("deal ID is empty")
	}

	if d.Price.IsZero() || d.Price.IsNaN() || d.Price.IsInf() == 1 {
		return errors.Errorf("price value is illegal: %v", d.Price)
	}

	if d.Volume.IsZero() || d.Volume.IsNaN() || d.Volume.IsInf() == 1 {
		return errors.Errorf("volume value is illegal: %v", d.Volume)
	}

	if !marketRegex.MatchString(d.Market) {
		return errors.Errorf(
			"invalid market %s, must be matches by: %s",
			d.Market,
			marketRegex.String(),
		)
	}

	if d.Time.Before(companyCreation) {
		return errors.Errorf("deal created time is to old: %v", d.Time)
	}

	return nil
}

func MustParseDecimal(num string) primitive.Decimal128 {
	if d, err := primitive.ParseDecimal128(num); err != nil {
		panic(err)
	} else {
		return d
	}
}
