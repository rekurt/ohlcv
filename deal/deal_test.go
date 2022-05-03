package deal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestService_GetLastTrades(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	mt.Run(
		"success", func(mt *mtest.T) {
			s := Service{DbCollection: mt.Coll}
			first := mtest.CreateCursorResponse(
				1, "foo.bar", mtest.FirstBatch, bson.D{
					{"_id", primitive.NewObjectID()},
					{"market", "ETH"},
				},
			)
			second := mtest.CreateCursorResponse(
				1, "foo.bar", mtest.NextBatch,
				bson.D{{"_id", primitive.NewObjectID()}, {"market", "BTC"}},
			)
			killCursors := mtest.CreateCursorResponse(
				0,
				"foo.bar",
				mtest.NextBatch,
			)
			mt.AddMockResponses(first, second, killCursors)

			trades, err := s.GetLastTrades(context.Background(), "sym", 1)
			require.NoError(t, err)
			assert.Len(t, trades, 2)
		},
	)
}
