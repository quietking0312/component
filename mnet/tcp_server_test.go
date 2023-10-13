package mnet

//func Test_NewTCPServer(t *testing.T) {
//	ln, err := net.Listen("tcp", "0.0.0.0:8888")
//	if err != nil {
//		t.Fatal(err)
//	}
//	var route = NewRouter()
//	c := func(message proto.Message) string {
//		msgType := reflect.TypeOf(message)
//		return strconv.FormatInt(int64(pb.C2S_value[strings.ToLower(msgType.Elem().Name())]), 10)
//	}
//	route.Register(c(&pb.Ping{}), func(msg Msg, a AgentIface) {
//		var req pb.Ping
//		if err := proto.Unmarshal(msg.Data, &req); err != nil {
//			t.Fatal(err)
//		}
//		fmt.Println(req.GetArgs())
//		a.Write(&pb.Pong{
//			Data: "go 1.20",
//		})
//	}, func(msg Msg, a AgentIface) {
//		fmt.Println("2222")
//	})
//	ser := NewTCPServer(65535, func(conn *TCPConn) AgentIface {
//		a := &Agent{conn: conn, log: _log, parser: &ProtoParser{}, handler: route}
//		return a
//	})
//	ser.Serve(ln)
//}
