package redis

import (
	"bufio"
	"bytes"
	. "launchpad.net/gocheck"
	"time"
)

type ClientSuite struct {
	c *Client
}

func init() {
	Suite(&ClientSuite{})
}

func (s *ClientSuite) SetUpTest(c *C) {
	var err error
	s.c, err = DialTimeout("tcp", "127.0.0.1:6379", time.Duration(10)*time.Second)
	c.Assert(err, IsNil)

	// select database
	r := s.c.Cmd("select", 8)
	c.Assert(r.Err, IsNil)
}

func (s *ClientSuite) TearDownTest(c *C) {
	s.c.Close()
}

func (s *ClientSuite) TestCmd(c *C) {
	v, _ := s.c.Cmd("echo", "Hello, World!").Str()
	c.Assert(v, Equals, "Hello, World!")
}

func (s *ClientSuite) TestPipeline(c *C) {
	s.c.Append("echo", "foo")
	s.c.Append("echo", "bar")
	s.c.Append("echo", "zot")

	v, _ := s.c.GetReply().Str()
	c.Assert(v, Equals, "foo")

	v, _ = s.c.GetReply().Str()
	c.Assert(v, Equals, "bar")

	v, _ = s.c.GetReply().Str()
	c.Assert(v, Equals, "zot")

	r := s.c.GetReply()
	c.Assert(r.Type, Equals, ErrorReply)
	c.Assert(r.Err, Equals, PipelineQueueEmptyError)
}

func (s *ClientSuite) TestParse(c *C) {
	parseString := func(b string) *Reply {
		s.c.reader = bufio.NewReader(bytes.NewBufferString(b))
		return s.c.parse()
	}

	// missing \n trailing
	r := parseString("foo")
	c.Check(r.Type, Equals, ErrorReply)
	c.Check(r.Err, NotNil)

	// error reply
	r = parseString("-ERR unknown command 'foobar'\r\n")
	c.Check(r.Type, Equals, ErrorReply)
	c.Check(r.Err.Error(), Equals, "ERR unknown command 'foobar'")

	// LOADING error
	r = parseString("-LOADING Redis is loading the dataset in memory\r\n")
	c.Check(r.Type, Equals, ErrorReply)
	c.Check(r.Err, Equals, LoadingError)

	// status reply
	r = parseString("+OK\r\n")
	c.Check(r.Type, Equals, StatusReply)
	c.Check(r.str, Equals, "OK")

	// integer reply
	r = parseString(":1337\r\n")
	c.Check(r.Type, Equals, IntegerReply)
	c.Check(r.int, Equals, int64(1337))

	// null bulk reply
	r = parseString("$-1\r\n")
	c.Check(r.Type, Equals, NilReply)

	// bulk reply
	r = parseString("$6\r\nfoobar\r\n")
	c.Check(r.Type, Equals, BulkReply)
	c.Check(r.str, Equals, "foobar")

	// null multi bulk reply
	r = parseString("*-1\r\n")
	c.Check(r.Type, Equals, NilReply)

	// multi bulk reply
	r = parseString("*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$6\r\nfoobar\r\n")
	c.Check(r.Type, Equals, MultiReply)
	c.Assert(len(r.Elems), Equals, 5)
	c.Check(r.Elems[0].int, Equals, int64(1))
	c.Check(r.Elems[1].int, Equals, int64(2))
	c.Check(r.Elems[2].int, Equals, int64(3))
	c.Check(r.Elems[3].int, Equals, int64(4))
	c.Check(r.Elems[4].str, Equals, "foobar")

	// invalid multi bulk reply
	r = parseString("*-2\r\n")
	c.Check(r.Type, Equals, ErrorReply)
	c.Check(r.Err, Equals, ParseError)

	// invalid reply
	r = parseString("@foo\r\n")
	c.Check(r.Type, Equals, ErrorReply)
	c.Check(r.Err, Equals, ParseError)
}
