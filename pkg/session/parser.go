package session

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"lion/pkg/guacd"
	"lion/pkg/logger"

	"github.com/jumpserver-dev/sdk-go/model"
	"github.com/jumpserver-dev/sdk-go/service"
)

var (
	charEnter = []byte("\r")
)

var _ ParseEngine = (*Parser)(nil)

type Parser struct {
	id            string
	jmsService    *service.JMService
	cmdRecordChan chan *ExecutedCommand

	buf bytes.Buffer

	inputPreState bool
	inputState    bool
	once          *sync.Once
	lock          *sync.RWMutex

	command       string
	cmdCreateDate time.Time

	closed            chan struct{}
	currentActiveUser CurrentActiveUser
}

func (p *Parser) initial() {
	p.once = new(sync.Once)
	p.lock = new(sync.RWMutex)
	p.closed = make(chan struct{})
	p.cmdRecordChan = make(chan *ExecutedCommand, 1024)
}

// ParseStream parses the data stream
func (p *Parser) ParseStream(userInChan chan *Message) {
	logger.Infof("Session %s: Parser start", p.id)
	go func() {
		defer func() {
			p.ParseUserInput(charEnter)
			// session ended, finalize the command result
			p.sendCommandRecord()
			close(p.cmdRecordChan)
			logger.Infof("Session %s: Parser routine done", p.id)
		}()
		maxTimeout := time.Second * 20
		cmdRecordTicker := time.NewTicker(time.Second * 30)
		defer cmdRecordTicker.Stop()
		lastActiveTime := time.Now()
		for {
			select {
			case <-p.closed:
				return
			case now := <-cmdRecordTicker.C:
				if now.Sub(lastActiveTime) > maxTimeout {
					p.ParseUserInput(charEnter) // manually finalize a command
				}
				continue
			case msg, ok := <-userInChan:
				if !ok {
					return
				}
				lastActiveTime = time.Now()
				p.UpdateActiveUser(msg)
				s := msg.Body
				var b []byte
				switch msg.Opcode {
				case guacd.InstructionMouse:
					var cmd string
					switch s[2] {
					case guacd.MouseLeft:
						cmd = "Left Button"
					case guacd.MouseRight:
						cmd = "Right Button"
					case guacd.MouseMiddle:
						cmd = "Middle Button"
					default:
						continue
					}
					p.ParseUserInput(charEnter) // manually finalize a command
					cmd = fmt.Sprintf("Mouse Position[%s,%s] %s\r", s[0], s[1], cmd)
					b = append(b, []byte(cmd)...)
				case guacd.InstructionKey:
					switch s[1] {
					case guacd.KeyPress:
						keyCode, err := strconv.Atoi(s[0])
						if err == nil {
							cb := []byte(guacd.KeysymToCharacter(keyCode))
							if len(cb) == 0 {
								// guacamole-common.js unicode calculation method
								// if (codepoint >= 0x0100 && codepoint <= 0x10FFFF)
								//      return 0x01000000 | codepoint;
								if keyCode > 0x01000000 {
									var to string
									unicode := strconv.FormatInt(int64(keyCode), 16)
									bs, _ := hex.DecodeString(unicode[3:])
									for i, bl, br, r := 0, len(bs), bytes.NewReader(bs), uint16(0); i < bl; i += 2 {
										_ = binary.Read(br, binary.BigEndian, &r)
										to += string(rune(r))
									}
									b = append(b, []byte(to)...)
								} else {
									// unknown key value, convert to rune character
									b = append(b, []byte(string(rune(keyCode)))...)
								}
							} else {
								b = append(b, cb...)
							}
						} else {
							b = append(b, []byte(guacd.KeyCodeUnknown)...)
						}
					default:
						continue
					}
				}
				if len(b) == 0 {
					continue
				}
				_, _ = p.WriteData(b)
				p.ParseUserInput(b)
			}
		}
	}()
}

// ParseUserInput parses the user's input
func (p *Parser) ParseUserInput(b []byte) {
	_ = p.parseInputState(b)
}

// parseInputState toggles the user input state, and finalizes commands and their results
func (p *Parser) parseInputState(b []byte) []byte {
	p.inputPreState = p.inputState
	if bytes.LastIndex(b, charEnter) >= 0 {
		// consecutive enter key input, finalize the result of the previous command if any
		p.sendCommandRecord()
		p.inputState = false
		// user pressed Enter, start finalizing the command
		p.parseCmdInput()
	} else {
		p.inputState = true
		// user started typing again, and was not in input state previously, start finalizing the previous command's result
		if !p.inputPreState {
			p.sendCommandRecord()
		}
	}
	return b
}

// parseCmdInput parses the command input
func (p *Parser) parseCmdInput() {
	command := p.Parse()
	if len(command) <= 0 {
		p.command = ""
	} else {
		p.command = command
	}
	p.cmdCreateDate = time.Now()
}

func (p *Parser) WriteData(b []byte) (int, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.buf.Len() >= 2048 {
		return 0, nil
	}
	if len(b) > 1 {
		p.buf.WriteByte(byte(' '))
	}
	return p.buf.Write(b)
}

func (p *Parser) Parse() string {
	line := p.buf.String()
	line = strings.TrimPrefix(line, string(charEnter))
	p.buf.Reset()
	return line
}

// Close closes the parser
func (p *Parser) Close() {
	select {
	case <-p.closed:
		return
	default:
		close(p.closed)
	}
	logger.Infof("Session %s: Parser close", p.id)
}

func (p *Parser) sendCommandRecord() {
	if p.command != "" {
		p.cmdRecordChan <- &ExecutedCommand{
			Command:     p.command,
			CreatedDate: p.cmdCreateDate,
			RiskLevel:   model.NormalLevel,
			User:        p.currentActiveUser,
		}
		p.command = ""
	}
}

func (p *Parser) CommandRecordChan() chan *ExecutedCommand {
	return p.cmdRecordChan
}

func (p *Parser) UpdateActiveUser(msg *Message) {
	p.currentActiveUser.UserId = msg.Meta.UserId
	p.currentActiveUser.User = msg.Meta.User
}

type ExecutedCommand struct {
	Command     string
	Output      string
	CreatedDate time.Time
	RiskLevel   int
	User        CurrentActiveUser
}

type CurrentActiveUser struct {
	UserId string
	User   string
}
