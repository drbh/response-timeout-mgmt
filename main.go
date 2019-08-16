package timeouter

import (
	"fmt"
	"time"

	"github.com/jasonlvhit/gocron"
	"golang.org/x/sync/syncmap"

	"github.com/looplab/fsm"

	"sync"
)

type MessageTimeoutManager struct {
	users syncmap.Map
}

var instance *MessageTimeoutManager
var once sync.Once

func GetInstance() *MessageTimeoutManager {
	once.Do(func() {
		instance = &MessageTimeoutManager{}
		instance.StartReminders()
	})
	return instance
}

type UserStateMachine struct {
	To           string
	Responded    bool
	WhenSent     time.Time
	WhenRecieved time.Time
	FSM          *fsm.FSM
	Timeout      int

	TimeSinceSent time.Duration

	Callback func(usm UserStateMachine)
}

func (d *UserStateMachine) enterState(e *fsm.Event) {
	fmt.Printf("The user %s is %s\n", d.To, e.Dst)
}

// type MessageTimeoutManager struct {
// 	users syncmap.Map
// }

func (a *MessageTimeoutManager) GetUserAndTriggerEvent(name, event string) {
	b, _ := a.users.Load(name)
	c := b.(UserStateMachine)
	c.FSM.Event(event)
}

func (a *MessageTimeoutManager) GetUser(name string) UserStateMachine {
	b, _ := a.users.Load(name)
	c := b.(UserStateMachine)
	return c
}

func (a *MessageTimeoutManager) AddCallback(name string, callback func(usm UserStateMachine)) {
	b, _ := a.users.Load(name)
	c := b.(UserStateMachine)
	c.Callback = callback
	a.users.Store(name, c)
}

func (a *MessageTimeoutManager) AddUser(name string, timeout int) {

	d := &UserStateMachine{
		To:        name,
		Responded: false,
		Timeout:   timeout,
	}

	var states = fsm.Events{
		{Name: "init", Src: []string{"start"}, Dst: "waiting_action"},
		{Name: "send_message", Src: []string{"waiting_action"}, Dst: "waiting_message_response"},
		{Name: "user_response", Src: []string{"waiting_message_response"}, Dst: "responded"},

		{Name: "done", Src: []string{"responded"}, Dst: "end"},
		{Name: "timeout", Src: []string{"waiting_action", "waiting_message_response"}, Dst: "end"},
	}

	d.FSM = fsm.NewFSM(
		"start",
		states,
		fsm.Callbacks{
			"after_event": func(e *fsm.Event) { d.enterState(e) },
			"send_message": func(e *fsm.Event) {
				v, _ := a.users.Load(name)
				c := v.(UserStateMachine)
				c.WhenSent = time.Now()
				a.users.Store(name, c)
			},

			"user_response": func(e *fsm.Event) {
				v, _ := a.users.Load(name)
				c := v.(UserStateMachine)
				c.WhenRecieved = time.Now()
				c.Responded = true
				c.TimeSinceSent = d.WhenRecieved.Sub(d.WhenSent)
				a.users.Store(name, c)
			},

			"end": func(e *fsm.Event) {
				v, _ := a.users.Load(name)
				c := v.(UserStateMachine)
				go c.Callback(c)
			},
		},
	)

	a.users.Store(name, *d)
}

func (a *MessageTimeoutManager) StartReminders() {
	go func() {
		gocron.Every(1).Seconds().Do(a.bar)
		<-gocron.Start()
	}()
}

func (a *MessageTimeoutManager) bar() {
	a.users.Range(func(key interface{}, value interface{}) bool {
		sched := value.(UserStateMachine)

		if !sched.WhenSent.IsZero() {
			now := time.Now()
			timeSinceSent := now.Sub(sched.WhenSent)
			seconds := time.Duration(sched.Timeout * 1000000000)

			if timeSinceSent > seconds {
				a.GetUserAndTriggerEvent(key.(string), "timeout")
			}

		}

		return true
	})

}
