package cocaine

import (
	"github.com/ugorji/go/codec"
	"fmt"
	"os"
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
	if apps := <-s.Find("apps", []string{"app"}); apps.Err != nil {
		t.Error("Storage.Find error", apps.Err)
	} else {
		t.Log(apps.Res)
	}

	if write := <-s.Write("GOTEST", "TEST", []byte("TESTDATA"), []string{"A", "B"}); write.Err != nil {
		t.Error("Fail")
	} else {
		t.Log("Write OK")
	}

	if manifest := <-s.Read("GOTEST", "TEST"); manifest.Err != nil {
		t.Error("Error", manifest.Err)
	} else {
		t.Log("Ok", manifest.Res)
	}
	if remove := <-s.Remove("GOTEST", "TEST2"); remove.Err != nil {
		t.Error(remove.Err)
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

	if enq := <-app.Enqueue("echo", []byte("PING")); enq.Err != nil {
		t.Error("Fail enqueu", enq.Err)
	} else {
		t.Log("OK")
	}
	if info := <-app.Info(); info.Err != nil {
		t.Error("Fail info", info.Err)
	} else {
		t.Log(info.Res)
	}

}

// method, url, version, headers, self._body
func TestReq(t *testing.T) {
	fmt.Println("START")
	file, err := os.Open("/Users/noxiouz/Documents/github/cocaine-framework-Go/REQ") // For read access.
	data := make([]byte, 1024)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(file)
	_, err = file.Read(data)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(UnpackProxyRequest(data))
}
