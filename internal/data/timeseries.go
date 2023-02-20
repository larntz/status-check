package data

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ResultCollection document format:
// {
//   timestamp: "unix_time??",
//   metadata: {"region": region, "check_id": checkID},
//   response_code: ResponseCode,
//   reoponse_time: ResponseTime,
//   response_desc: ResponseText
// }

// CreateResultCollection creates a time series collection to store check results
func CreateResultCollection(ctx context.Context, client *mongo.Client, name string) {
	db := client.Database("status")
	tso := options.TimeSeries().SetTimeField("timestamp").SetMetaField("metadata").SetGranularity("minutes")
	// expire after 3 days (60*60*24*3 = 259200)
	opts := options.CreateCollection().SetTimeSeriesOptions(tso).SetExpireAfterSeconds(259200)
	db.CreateCollection(ctx, name, opts)
}
