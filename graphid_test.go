package ag

import (
	"bytes"
	"testing"
)

func TestNewGraphIdError(t *testing.T) {
	tests := []string{
		"",
		"0.1",
		"1.0",
		"65536.281474976710655",
		"65535.281474976710656",
	}
	for _, s := range tests {
		_, err := NewGraphId(s)
		if err == nil {
			t.Errorf("error expected for %q", s)
		}
	}
}

func mustNewGraphId(str string) GraphId {
	gid, err := NewGraphId(str)
	if err != nil {
		panic(err)
	}
	return gid
}

func TestGraphIdEqual(t *testing.T) {
	tests := []struct {
		x     GraphId
		y     GraphId
		equal bool
	}{
		{mustNewGraphId("NULL"), mustNewGraphId("1.1"), false},
		{mustNewGraphId("1.1"), mustNewGraphId("NULL"), false},
		{mustNewGraphId("1.1"), mustNewGraphId("1.1"), true},
		{mustNewGraphId("1.1"), mustNewGraphId("65535.281474976710655"), false},
	}
	for _, c := range tests {
		equal := c.x.Equal(c.y)
		if equal != c.equal {
			t.Errorf("got %q.Equal(%q) == %t, want %t", c.x, c.y, equal, c.equal)
		}
	}
}

func TestGraphIdScanNil(t *testing.T) {
	var gid GraphId
	err := gid.Scan(nil)
	if err != nil {
		t.Fatal(err)
	}
	if gid.Valid {
		t.Errorf("got %q, want NULL", gid)
	}
}

func TestGraphIdScanType(t *testing.T) {
	src := 0
	var gid GraphId
	err := gid.Scan(src)
	if err == nil {
		t.Errorf("error expected for %T", src)
	}
}

func TestGraphIdScanDuplicate(t *testing.T) {
	src := []byte("1.1")

	var gid GraphId
	_ = gid.Scan(src)

	if &gid.b[0] == &src[0] {
		t.Error("GraphId references underlying array")
	}
}

func TestGraphIdArrayScanType(t *testing.T) {
	src := 0
	var gids []GraphId
	a := Array(&gids)
	err := a.Scan(src)
	if err == nil {
		t.Errorf("error expected for %T", src)
	}
}

var graphIdArrayTests = []struct {
	val  interface{}
	gids []GraphId
}{
	{
		[]byte("{NULL,1.1,65535.281474976710655}"),
		[]GraphId{
			mustNewGraphId("NULL"),
			mustNewGraphId("1.1"),
			mustNewGraphId("65535.281474976710655"),
		},
	},
	{
		[]byte("{}"),
		[]GraphId{},
	},
	{
		nil,
		nil,
	},
}

func TestGraphIdArrayScan(t *testing.T) {
	for _, c := range graphIdArrayTests {
		var gids []GraphId
		a := Array(&gids)
		err := a.Scan(c.val)
		if err != nil {
			t.Error(err)
			continue
		}

		if len(gids) != len(c.gids) {
			t.Errorf("got len(gids) == %d, want %d", len(gids), len(c.gids))
			continue
		}

	EqualLoop:
		for i, gid := range gids {
			if !gid.Valid && !c.gids[i].Valid {
				continue
			}
			if !gid.Equal(c.gids[i]) {
				t.Errorf("got %q, want %q", gid, c.gids[i])
				break EqualLoop
			}
		}
	}
}

func TestGraphIdArrayValue(t *testing.T) {
	for _, c := range graphIdArrayTests {
		val, err := Array(c.gids).Value()
		if err != nil {
			t.Error(err)
			continue
		}

		if val == nil {
			if c.val != nil {
				t.Errorf("error expected for %v", c.gids)
			}
		} else {
			b, ok := val.([]byte)
			if !ok {
				t.Errorf("got %T, want []byte", val)
				continue
			}
			if !bytes.Equal(b, c.val.([]byte)) {
				t.Errorf("got %q, want %q", string(b), string(c.val.([]byte)))
			}
		}
	}
}

func TestServerGraphId(t *testing.T) {
	skipUnlessServerTest(t)

	db := mustOpenAndSetGraph(t)
	defer db.Close()

	_, err := db.Exec(`CREATE (:gid)`)
	if err != nil {
		t.Fatal(err)
	}

	var gid GraphId
	err = db.QueryRow(`MATCH (n:gid) RETURN id(n)`).Scan(&gid)
	if err != nil {
		t.Fatal(err)
	}
	if !gid.Valid {
		t.Error("got NULL, want Valid GraphId")
	}

	var cnt int64
	q := `MATCH (n:gid) WHERE id(n) = $1 RETURN count(*)`
	err = db.QueryRow(q, gid).Scan(&cnt)
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Errorf("got %d, want %d", cnt, 1)
	}

	gids := []GraphId{mustNewGraphId("NULL"), mustNewGraphId("1.1"), mustNewGraphId("65535.281474976710655")}
	var gidsOut []GraphId
	err = db.QueryRow(`SELECT $1::_graphid`, Array(gids)).Scan(Array(&gidsOut))
	if err != nil {
		t.Fatal(err)
	}
	for i, gid := range gidsOut {
		if !gid.Valid && !gids[i].Valid {
			continue
		}
		if !gid.Equal(gids[i]) {
			t.Errorf("got %q, want %q", gid, gids[i])
			break
		}
	}

	err = db.QueryRow(`SELECT NULL::_graphid`).Scan(Array(&gidsOut))
	if err != nil {
		t.Fatal(err)
	}
	if gidsOut != nil {
		t.Errorf("got %v, want nil", gidsOut)
	}
}
