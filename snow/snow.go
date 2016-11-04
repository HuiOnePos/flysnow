package snow

import (
	"flysnow/models"
	"flysnow/utils"
	"fmt"
	"labix.org/v2/mgo/bson"
	"strconv"
	"sync"
)

type SnowSys struct {
	*utils.SnowKey
	RedisConn *utils.RedisConn
	Tag, Term string
	Now       int64
}

var snowlock rwmutex

type rwmutex struct {
	//m map[string]*sync.RWMutex
	l *sync.Mutex
}

func init() {
	//snowlock = rwmutex{m: map[string]*sync.RWMutex{}}
	snowlock = rwmutex{l: new(sync.Mutex)}
}

type SnowData struct {
	Key   string                   `json:"s_key" bson:"s_key"`
	STime int64                    `json:"s_time" bson:"s_time"`
	ETime int64                    `json:"e_time" bson:"e_time"`
	Data  []map[string]interface{} `json:"data" bson:"data"`
	Index map[string]interface{}
	Term  string
	Tag   string
}

func NeedRotate(snowsys *SnowSys, snow models.Snow) (bl bool) {
	now := snowsys.Now
	b, _ := snowsys.RedisConn.Dos("HGET", snowsys.Key, "e_time")
	if b != nil {
		endt, _ := strconv.ParseInt(string(b.([]byte)), 10, 64)
		if endt < now {
			bl = true
			snowlock.l.Lock()
			snowsys.RedisConn.Dos("RENAME", snowsys.Key, snowsys.Key+"_rotate")
			end := utils.DurationMap[snow.InterValDuration](now, snow.Interval)
			start := utils.DurationMap[snow.InterValDuration+"l"](end, snow.Interval)
			snowsys.RedisConn.Dos("HMSET", snowsys.Key, "s_time", start, "e_time", end)
			snowlock.l.Unlock()
		} else {
			return
		}
	}
	end := utils.DurationMap[snow.InterValDuration](now, snow.Interval)
	start := utils.DurationMap[snow.InterValDuration+"l"](end, snow.Interval)
	snowsys.RedisConn.Dos("HMSET", snowsys.Key, "s_time", start, "e_time", end)
	utils.Log.Error(snowsys.Key, start, end)
	return

}

func Rotate(snowsys *SnowSys, snows []models.Snow) {
	snowsys.RedisConn = utils.NewRedisConn(snowsys.Tag)
	defer snowsys.RedisConn.Close()
	tag := snowsys.Tag
	term := snowsys.Term
	if len(snows) == 0 || !NeedRotate(snowsys, snows[0]) {
		return
	}
	b, _ := snowsys.RedisConn.Dos("HGETALL", snowsys.Key+"_rotate")
	if b == nil {
		return
	}
	tb := b.([]interface{})
	if len(tb) == 0 {
		return
	}
	snowsys.RedisConn.Dos("DEL", snowsys.Key+"_rotate")
	go func() {
		dm := map[string]interface{}{}
		for i := 0; i < len(tb); i = i + 2 {
			dm[string(tb[i].([]uint8))], _ = strconv.ParseInt(string(tb[i+1].([]uint8)), 10, 64)
		}
		now := snowsys.Now
		session := utils.MgoSessionDupl(tag)
		mc := session.DB(models.MongoDT + tag).C(term)
		defer session.Close()
		var data SnowData
		var lasttime int64
		retatedata := []map[string]interface{}{}
		for sk, s := range snows {
			key := snowsys.Key + "_" + fmt.Sprintf("%d", s.Interval) + "_" + s.InterValDuration
			if sk == 0 {
				mc.Find(bson.M{"s_key": key}).One(&data)
				data.ETime = dm["e_time"].(int64)
				data.STime = utils.DurationMap[s.TimeoutDuration+"l"](data.ETime, s.Timeout)
				td := []map[string]interface{}{}
				data.Data = append(data.Data, dm)
				retatedata = data.Data
				lasttime = data.STime
				for k, v := range data.Data {
					if d, ok := v["s_time"]; ok {
						if d.(int64) >= data.STime {
							td = data.Data[k:]
							retatedata = data.Data[:k]
							break
						}
					}
				}
				mc.Upsert(bson.M{"s_key": key}, bson.M{"$set": bson.M{"s_time": data.STime, "e_time": data.ETime, "tag": tag, "term": term, "data": td, "index": snowsys.Index}})
				if len(retatedata) == 0 {
					break
				}
			} else {
				data = SnowData{}
				mc.Find(bson.M{"s_key": key}).One(&data)
				data.ETime = lasttime
				data.STime = utils.DurationMap[s.TimeoutDuration+"l"](data.ETime, s.Timeout)
				lasttime = data.STime
				ttt := retatedata
				td := []map[string]interface{}{}
				retatedata = data.Data
				for k, v := range data.Data {
					if d, ok := v["s_time"]; ok {
						if d.(int64) >= data.STime {
							td = data.Data[k:]
							retatedata = data.Data[:k]
							break
						}
					}
				}

				for _, v := range ttt {
					o := false
					tmpsnow := snows[sk]
					v["e_time"] = utils.DurationMap[tmpsnow.InterValDuration](v["e_time"].(int64), tmpsnow.Interval)
					v["s_time"] = utils.DurationMap[tmpsnow.InterValDuration+"l"](v["e_time"].(int64), tmpsnow.Interval)
					lasttime = v["e_time"].(int64)
					for k1, v1 := range td {
						if v["s_time"].(int64) >= v1["s_time"].(int64) && v["e_time"].(int64) <= v1["e_time"].(int64) {
							for tk, tv := range v {
								if tk != "s_time" && tk != "e_time" {
									if v2, ok := v1[tk]; ok {
										v1[tk] = v2.(int64) + tv.(int64)
									} else {
										v1[tk] = tv
									}
								}
							}
							td[k1] = v1
							o = true
						}
					}
					if !o {
						if v["s_time"].(int64) >= data.STime {
							td = append(td, v)
						} else {
							retatedata = append(retatedata, v)
						}

					}
				}
				mc.Upsert(bson.M{"s_key": key}, bson.M{"$set": bson.M{"s_time": data.STime, "e_time": data.ETime, "tag": tag, "term": term, "data": td, "index": snowsys.Index}})
				if len(retatedata) == 0 {
					break
				}
			}
		}
		if len(retatedata) > 0 {
			tmp := bson.M{}
			for _, v := range retatedata {
				for k1, v1 := range v {
					if k1 == "s_time" || k1 == "e_time" {
						continue
					}
					if v2, ok := tmp[k1]; ok {
						tmp[k1] = v2.(int64) + v1.(int64)
					} else {
						tmp[k1] = v1
					}
				}
			}
			mc.Upsert(bson.M{"s_key": snowsys.Key}, bson.M{"$inc": tmp, "$set": bson.M{
				"e_time": now, "tag": tag, "term": term, "index": snowsys.Index}})

		}
	}()
}
