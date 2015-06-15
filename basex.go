package basex

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net"
	"strings"
)

type BaseXClient struct {
	*bufio.ReadWriter
	con net.Conn
}

func (b *BaseXClient) exec(cmd byte, arg string) {
	b.ReadWriter.WriteByte(cmd)
	b.send(arg)
}

func (b *BaseXClient) send(str string) {
	strLen := len(str)
	for i := 0; i < strLen; i++ {
		if str[i] == 0 || str[i] == 255 {
			b.ReadWriter.WriteByte(0xFF)
		}
		b.ReadWriter.WriteByte(str[i])
	}
	b.WriteByte(0)
	b.ReadWriter.Flush()
}

func (b *BaseXClient) ok() bool {
	d, _ := b.ReadWriter.ReadByte()
	return d == 0
}

func New(adr string, user string, pass string) (cli *BaseXClient, err error) {
	cli = &BaseXClient{}

	cli.con, _ = net.Dial("tcp", adr)
	cli.ReadWriter = bufio.NewReadWriter(cli.con, cli.con)
	ts := cli.ReadString()

	var ok bool
	cli.send(user)
	if i := strings.Index(ts, ":"); i != -1 {
		ok = cli.login(user, pass, string(ts[:i]), string(ts[i+1:]))
	} else {
		ok = cli.loginLegacy(pass, ts)
	}

	if ok {
		err = errors.New("Login error")
		cli = nil
	}

	return
}

func (b *BaseXClient) Close() {
	b.con.Close()
}

func (b *BaseXClient) Command(cmd string) (string, string) {
	b.send(cmd)
	result := b.ReadString()
	info := b.ReadString()
	b.ok()
	return result, info
}

func (b *BaseXClient) Query(qry string) *Query {
	b.exec(0, qry)
	id := b.ReadString()
	if !b.ok() {
		panic(b.ReadString())
	}
	return &Query{id: id, cli: b, hasNext: false, lastResult: "", state: 0}
}

func (b *BaseXClient) WriteString(str string) {
	b.ReadWriter.WriteString(str)
	b.ReadWriter.WriteByte(0)
	b.ReadWriter.Flush()
}

func (b *BaseXClient) WriteByte(bte byte) {
	b.ReadWriter.WriteByte(bte)
	b.Flush()
}

func (b *BaseXClient) ReadString() string {
	buf := bytes.NewBuffer(nil)
	for {
		d, err := b.ReadWriter.ReadByte()

		if err != nil || d == 0 {
			break
		}

		if d == 255 {
			d, err = b.ReadWriter.ReadByte()
		}

		buf.WriteByte(d)
	}
	return buf.String()
}

func (this *BaseXClient) login(user, password, realm, nonce string) bool {
	this.send(md5Hex(md5Hex(user+":"+realm+":"+password) + nonce))
	return this.ok()
}

func (this *BaseXClient) loginLegacy(password, nonce string) bool {
	this.send(md5Hex(md5Hex(password) + nonce))
	return this.ok()
}

func md5Hex(str string) string {
	hash := md5.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil))
}
