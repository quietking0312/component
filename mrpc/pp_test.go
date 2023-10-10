package mrpc

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/quietking0312/component/mcyptos"
	pb "github.com/quietking0312/component/mrpc/proto"
	"testing"
	"time"
)

var Data = &pb.Data{
	Name:  "hello",
	Age:   15,
	Group: 568,
	Item:  []int32{8888, 6666, 3333, 6666, 111111, 323142141, 56131},
	Prop: []*pb.Prop{
		{Item: 55, Type: 5, Count: 555},
		{Item: 56, Type: 4, Count: 5553354},
		{Item: 58, Type: 3, Count: 555533},
		{Item: 52, Type: 2, Count: 555233},
		{Item: 25, Type: 1, Count: 555452},
		{Item: 35, Type: 10, Count: 5564544},
	},
}

func Test_JSON(t *testing.T) {
	d := map[string]interface{}{
		"code": 0,
		"data": Data,
	}
	dBytes, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("len", len(dBytes))
	t1 := time.Now()
	for i := 0; i < 10000; i++ {
		var d0 = make(map[string]interface{})
		err = json.Unmarshal(dBytes, &d0)
		if err != nil {
			t.Fatal(err)
		}
	}
	dt := time.Now().Sub(t1)
	fmt.Println(dt.Nanoseconds())
}

func Test_PROTO(t *testing.T) {
	dataBytes, err := proto.Marshal(Data)
	if err != nil {
		t.Fatal(err)
	}
	d := map[string]interface{}{
		"code": 0,
		"data": mcyptos.EncodeBase64(dataBytes),
	}
	dBytes, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("len", len(dBytes))
	t1 := time.Now()
	for i := 0; i < 10000; i++ {
		var d0 = make(map[string]interface{})
		err = json.Unmarshal(dBytes, &d0)
		if err != nil {
			t.Fatal(err)
		}
		d0DataBytes, err := mcyptos.DecodeBase64(d0["data"].(string))
		if err != nil {
			t.Fatal(err)
		}
		var d1 = pb.Data{}
		err = proto.Unmarshal(d0DataBytes, &d1)
		if err != nil {
			t.Fatal(err)
		}
	}
	dt := time.Now().Sub(t1)
	fmt.Println(dt.Nanoseconds())
}
