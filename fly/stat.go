package fly

import "flysnow/snow"

type Statistics struct {
}

func (s *Statistics) reader(d *BodyData) {

	err, result := snow.Stat(d.Body, d.Tag)
	if err != nil {
		log.ERROR.Printf("Stat error tag:%s,err:%s", d.Tag, err)
	}
	ConnRespChannel <- &connResp{d.Connid, 0, result}
}
