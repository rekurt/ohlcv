'use admin'

db.createUser({
    user: '$MONGODB_USER',
    pwd:  '$MONGODB_PASSWORD',
    roles: [{
        role: 'readWrite',
        db:   '$MONGODB_NAME'
    }]
})

db.createCollection(
    '$MONGODB_DEAL_COLLECTION_NAME',
    {
        timeseries: {
            timeField: "time",
            metaField: "market",
            granularity: "minutes"
        }
    }
)

db['$MONGODB_NAME'].createIndex({"deal_id": 1 }, {unique: true})
db['$MONGODB_NAME'].createIndex({"time": -1 })
