// HTTP-message   = start-line CRLF *( field-line CRLF ) CRLF [ message-body ]
// start-line     = request-line | status-line

// request-line   = method SP request-target SP HTTP-version

// method         = token
// request-target = origin-form | absolute-form | authority-form | asterisk-form
// origin-form    = absolute-path [ "?" query ]
// absolute-form  = absolute-URI
// authority-form = uri-host ":" port
// asterisk-form  = "*"

// status-line = HTTP-version SP status-code SP [ reason-phrase ]

// HTTP-version  = HTTP-name "/" DIGIT "." DIGIT 
// HTTP-name     = %s"HTTP"
// status-code    = 3DIGIT
// reason-phrase  = 1*( HTAB / SP / VCHAR / obs-text )

// field-line   = field-name ":" OWS field-value OWS


// // obs-fold     = OWS CRLF RWS
// //				; obsolete line folding

// message-body = *OCTET
// Transfer-Encoding = #transfer-coding

// chunked-body   = *chunk last-chunk trailer-section CRLF
// chunk          = chunk-size [ chunk-ext ] CRLF chunk-data CRLF
// chunk-size     = 1*HEXDIG
// last-chunk     = 1*("0") [ chunk-ext ] CRLF
// chunk-data     = 1*OCTET

// chunk-ext      = *( BWS ";" BWS chunk-ext-name [ BWS "=" BWS chunk-ext-val ] )
// chunk-ext-name = token
// chunk-ext-val  = token / quoted-string
// trailer-section   = *( field-line CRLF )


package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)


type mock struct {
	current int;
	buffer []uint8
}

func (self *mock) read(buff []uint8) int {
	i := self.current;
	j := 0;
	for ; j < len(buff) && i < len(self.buffer); i++ {
		buff[j] = self.buffer[i];
		j++;
	}
	self.current = i;
	return j;
}

func (self *mock) readLine(buff []uint8) int {
	var i = self.current;
	for  {
		if i - self.current >= len(buff) || i >= len(self.buffer) || self.buffer[i] == '\n' {
			break
		}
		buff[i-self.current] = self.buffer[i];
		i+=1;
	}
	read_bytes := i - self.current;
	self.current = i+1;
	return read_bytes;
}


func initMockBuffer(str string) mock {
	return mock{
		current: 0, buffer: []byte(str),
	}
}

func memset(buff []uint8, c uint8) {
	for i:= range buff{
		buff[i] = c
	}
}

type HTTPMethod = string;

const (
	GET   HTTPMethod 	= "GET";
	POST  HTTPMethod 	= "POST";
)

type MessageType int

const (
	REQUEST MessageType = 0
	RESPONSE MessageType = 1
)

type Message struct {
	message_type MessageType;
	http_version string;
	method HTTPMethod;
	headers map[string][]string;
}

func InitMessage() Message {
	var message Message;
	message.headers = make(map[string][]string);
	return message;
}

func startsWith(byte_array []byte, word string) bool {
	if len(word) > len(byte_array) {
		return false;
	}
	return bytes.Equal(byte_array[:len(word)], []byte(word));
}

func main()  {
	var me = "GET /hello.txt HTTP/1.1\r\ntransfer-encoding: chunked\r\n\r\n5\r\nabcde\r\n0\r\n"
	//var message = "a\nb\nc\n"
	//var me = "HTTP/1.1 200 OK\r\n"
	var mock = initMockBuffer(me);
	var buff [1024]uint8
	var message = InitMessage();
	message.headers = make(map[string][]string);
	regex := regexp.MustCompile(`HTTP\/[0-9]\.[0-9]`);
	regex_comma := regexp.MustCompile("[ ]*,[ ]*");

	for i := 0; ; i++  {
		var read_bytes = mock.readLine(buff[:]);
		if read_bytes == 0 { break }
		var read_line = buff[:read_bytes];
		if read_line[len(read_line)-1] != '\r' {
			panic("Expect line feed")
		}
		read_line = buff[:read_bytes-1]
		if len(read_line) == 0 {
			break;
		}
		if i == 0 {
			if startsWith(buff[:], "HTTP") {
				message.message_type = RESPONSE;
				var found_buff, rem, found = bytes.Cut(read_line[:read_bytes], []byte(" "))
				if !found {
					panic("expect HTTP sp")
				}
				if !regex.Match(found_buff) {
					panic("invalid http version string")
				}
				found_buff, rem, found = bytes.Cut(rem[:], []byte(" "));
				if !found {
					panic("expect Code sp")
				}
				found_buff, rem, found = bytes.Cut(rem[:], []byte("\r"));
				if len(found_buff) == 0 {
					panic("expect reason")
				}
			} else {
				message.message_type = REQUEST
				var found_buff, rem, found = bytes.Cut(read_line[:read_bytes], []byte(" "));
				if !found {
					panic("expect method sp");
				}
				found_buff, rem, found = bytes.Cut(rem[:], []byte(" "));
				if !found {
					panic("expect uri sp")
				}
				found_buff, rem, found = bytes.Cut(rem[:], []byte("\r"));
				if len(found_buff) == 0 {
					panic("expect HTTP version")
				}
				if !regex.Match(found_buff) {
					panic("invalid http version string")
				}
			}
		} else {
			var field_name, rem , found = bytes.Cut(read_line, []byte(":"));

			if !found {
				panic("expect field name");
			}
			if field_name[len(field_name)-1] == ' ' {
				panic("Unexpected space");
			}
			rem = bytes.TrimSpace(rem)
			field_values := regex_comma.Split(string(rem), -1);
			message.headers[strings.ToLower(string(field_name))] = field_values;
		}
	}
	content_len_value := message.headers["content-length"];
	if content_len_value != nil {
		content_len, err := strconv.ParseUint(content_len_value[0], 10, 32)
		if err != nil {
			panic("content len parse failed");
		}
		Normal(mock, uint32(content_len))
		return
	} else {
		transfer_encoding := message.headers["transfer-encoding"]
		if transfer_encoding != nil {
			if transfer_encoding[len(transfer_encoding)-1] == "chunked" {
				Chunked(mock);
				return
			}
		}
	}
	panic("unsupported encoding");
}


func Chunked(mock mock) {
	var buff [1024]uint8;
	for {
		read := mock.readLine(buff[:]);
		if read == 0 {
			panic("unexpected end of stream")
		}
		if buff[read-1] != '\r' {
			panic("expect to end with '\r'")
		}
		expectedlen, err := strconv.ParseInt(string(buff[:read-1]), 10, 64)
		if err != nil {
			panic(fmt.Sprintf("failed to parse len %s", buff[:read]))
		}
		if expectedlen == 0 {
			fmt.Println("end of stream") 
			if mock.current != int(len(mock.buffer)) {
				panic("stream end with error");
			}
			mock.readLine(buff[:]);
			break
		} 
		var read_bytes = 0;
		var buff [1024]uint8;
		for {
			r := mock.read(buff[:expectedlen])
			read_bytes += r;
			fmt.Println("read -> ", string(buff[:]))
			if read_bytes >= int(expectedlen) || r == 0 {
				break
			} 
		}
		if read_bytes != int(expectedlen) {
			panic(fmt.Sprintf("unexpected length of chunk %d %d", expectedlen,read_bytes));
		}
		read = mock.read(buff[:2]);
		if (!bytes.HasPrefix(buff[:], []byte{'\r', '\n'})) {
			panic("expect \\r\\n");
		}
	}
}

func Normal(mock mock, len uint32) {
	if len == 0 { return }
	var buff [1024]uint8;
	read := 0; 
	for {
		r := mock.read(buff[:])
		read += r;
		if read >= int(len) || r == 0 {
			break
		}
	}
	if read != int(len) {
		panic("early end of stream");
	}
}

