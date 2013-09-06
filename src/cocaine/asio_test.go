package cocaine

import (
	//"bytes"
	"codec"
	"testing"
)

// Test unpacker in case of partial feeding
func TestStreamUnpackerPartialFeeder(t *testing.T) {
	//TEST()
	var (
		mh codec.MsgpackHandle
		h  = &mh
	)
	var buf []byte
	enc := codec.NewEncoderBytes(&buf, h)
	enc.Encode([]interface{}{6, 2, []interface{}{}})
	t.Logf("Buff %s, %d", buf, len(buf))

	Unpacker := NewStreamUnpacker()
	if len(Unpacker.Feed(buf[:1])) != 0 {
		t.Error("Unpack message from invalid data")
	}
	if len(Unpacker.Feed(buf[1:])) != 1 {
		t.Error("Didn't unpack message from valid buffer")
	}
}

func TestStreamUnpackerExtraFeeder(t *testing.T) {
	var (
		mh codec.MsgpackHandle
		h  = &mh
	)
	var buf []byte
	enc := codec.NewEncoderBytes(&buf, h)
	enc.Encode([]interface{}{6, 2, []interface{}{}})
	t.Logf("Buff %s, %d", buf, len(buf))

	buf = append(buf, []byte("sadsds")...)
	Unpacker := NewStreamUnpacker()
	if len(Unpacker.Feed(buf)) == 0 {
		t.Error("Unpack message from valid data")
	}
}

func TestLocator(t *testing.T) {
	l := NewLocator("localhost", 10053)
	r := <-l.Resolve("GO")
	t.Log(r.API)
	r = <-l.Resolve("GO")
	t.Log(r.API)
}

func TestStorage(t *testing.T) {
	s, err := NewStorage("localhost", 10053)
	t.Log(s)
	if err != nil {
		t.Error("Unable to create storage", err)
	}
	// Find all apps
	if apps := <-s.Find("apps", []string{"app"}); apps.err != nil {
		t.Error("Storage.Find error", apps.err)
	} else {
		t.Log(apps.Res)
	}

	if write := <-s.Write("GOTEST", "TEST", []byte("TESTDATA"), []string{"A", "B"}); write.err != nil {
		t.Error("Fail")
	} else {
		t.Log("Write OK")
	}

	if manifest := <-s.Read("GOTEST", "TEST"); manifest.err != nil {
		t.Error("Error", manifest.err)
	} else {
		t.Log("Ok", manifest.Res)
	}
	if remove := <-s.Remove("GOTEST", "TEST2"); remove.err != nil {
		t.Error(remove.err)
	} else {
		t.Log("Remove OK")
	}
}

func TestApplication(t *testing.T) {
	app, err := NewApplication("GO", "localhost", 10053)
	t.Log(app)
	if err != nil {
		t.Error("Unable to create app", err)
	}

	if enq := <-app.Enqueue("echo", []byte("PING")); enq.err != nil {
		t.Error("Fail enqueu", enq.err)
	} else {
		t.Log("OK")
	}
	if info := <-app.Info(); info.err != nil {
		t.Error("Fail info", info.err)
	} else {
		t.Log(info.Res)
	}

}
